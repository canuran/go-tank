package main

import (
	"bytes"
	_ "embed"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	audio2 "github.com/hajimehoshi/ebiten/v2/examples/resources/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"image"
	"image/color"
	"io"
	"log"
	"math"
	"math/rand"
	"strconv"
)

var (
	//go:embed hit.ogg
	HitSound []byte
	//go:embed explode.ogg
	ExplodeSound []byte
	//go:embed chsfont.ttf
	ChsFont     []byte
	GamepadID   ebiten.GamepadID
	AudioCtx    = audio.NewContext(48000)
	GroundTrees = [][2]float64{{-0.01, -0.02}, {0.1, 1}, {0.2, 0.25}, {0.45, 0.65}, {0.6, 0.12}, {1, 0.4}}
	LifeColors  = []color.RGBA{colornames.Orangered, colornames.Yellow, colornames.Aliceblue}
	TankAngles  = []float64{AngleZero, AngleHalfPi, AnglePi, AngleTrebleHalfPi}

	// TankNames 第1个是玩家坦克
	TankNames    = []string{"tank_sand", "tank_dark", "tank_green", "tank_red", "tank_blue"}
	TankSpeeds   = []float64{8, 3, 4, 5, 6}
	BulletSpeeds = []float64{32, 4, 5, 6, 7}
	BulletNames  = []string{"bulletSand1_outline", "bulletDark1_outline", "bulletGreen1_outline", "bulletRed1_outline", "bulletBlue1_outline"}
)

func main() {
	g := &Game{title: "坦克大战", width: 1200, height: 900}
	g.spriteImages = LoadSpritesImage()
	g.spritesInfos = LoadSpriteInfos()

	chsFont, err := opentype.Parse(ChsFont)
	FatalIfError(err)
	g.chsFont, err = opentype.NewFace(chsFont,
		&opentype.FaceOptions{
			Size: 20, DPI: 72,
			Hinting: font.HintingVertical,
		})
	FatalIfError(err)

	g.groundAudio = newInfinitePlayer(bytes.NewReader(audio2.Ragtime_ogg))
	g.groundAudio.Play()
	g.hitAudio = newPlayer(bytes.NewReader(HitSound))
	g.hitAudio.SetVolume(0.4)
	g.explodeAudio = newPlayer(bytes.NewReader(ExplodeSound))
	g.explodeAudio.SetVolume(0.6)
	g.initGround()
	g.Restart()

	ebiten.SetWindowTitle(g.title)
	ebiten.SetWindowSize(g.width, g.height)
	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetWindowIcon([]image.Image{g.getIconImage()})
	err = ebiten.RunGame(g)
	FatalIfError(err)
}

type Game struct {
	title         string
	width         int
	height        int
	spriteImages  *ebiten.Image
	spritesInfos  map[string]SpriteInfo
	outputSprites bool
	hitAudio      *audio.Player
	explodeAudio  *audio.Player
	groundAudio   *audio.Player
	chsFont       font.Face
	ground        *Ground
	hero          *Hero
	enemy         *Chain[*Enemy]
	updates       int
	score         int
	highScore     int
	pause         bool
	pauseCool     int
	restartCool   int
}

type Ground struct {
	game   *Game
	grass1 SpriteInfo
	grass2 SpriteInfo
	trees  *Chain[*BoxSprite]
}

func (g *Game) Update() error {
	gamepadIDs := inpututil.AppendJustConnectedGamepadIDs([]ebiten.GamepadID{})
	for _, gamepadID := range gamepadIDs {
		GamepadID = gamepadID
		break
	}

	if g.pauseCool < 30 {
		g.pauseCool++
	} else if ebiten.IsKeyPressed(ebiten.KeySpace) ||
		ebiten.IsStandardGamepadButtonPressed(GamepadID, ebiten.StandardGamepadButtonCenterRight) {
		g.pauseCool = 0
		g.pause = !g.pause
	}
	if g.restartCool < 30 {
		g.restartCool++
	} else if ebiten.IsKeyPressed(ebiten.KeyR) ||
		ebiten.IsStandardGamepadButtonPressed(GamepadID, ebiten.StandardGamepadButtonCenterLeft) {
		g.Restart()
	}
	if g.pause {
		return nil
	}

	g.hero.UpdateMove()
	g.hero.UpdateShoot()
	if g.pause {
		return nil
	}

	for enemy := g.enemy; enemy != nil; enemy = enemy.Next {
		enemy.Value.AutoMove()
		enemy.Value.AutoShoot()
	}
	g.updates++
	return nil
}

func (g *Game) Restart() {
	g.updates = 0
	g.restartCool = 0
	g.pause = true
	g.score = 0
	g.initHero()
	g.initEnemies()
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.ground.Draw(screen)
	text.Draw(screen, "得分："+strconv.Itoa(g.score), g.chsFont, 3, 22, colornames.Aliceblue)
	text.Draw(screen, "最高："+strconv.Itoa(g.highScore), g.chsFont, 3, 45, colornames.Aliceblue)
	fps := "FPS：" + strconv.Itoa(int(ebiten.ActualFPS()))
	text.Draw(screen, fps, g.chsFont, g.width-len(fps)*10, 22, colornames.Aliceblue)

	desc := "空格键暂停，R键重开，WSAD或方向键移动，Ctrl或Enter键攻击，支持手柄"
	if g.pause {
		desc = "空格键开始，R键重开，WSAD或方向键移动，Ctrl或Enter键攻击，支持手柄"
	}
	text.Draw(screen, desc, g.chsFont, 230, 22, colornames.Aliceblue)

	for enemy := g.enemy; enemy != nil; enemy = enemy.Next {
		enemy.Value.Draw(screen)
	}
	g.hero.Draw(screen)
	if g.outputSprites {
		g.OutputSpriteInfos()
		g.outputSprites = false
	}
}

func (g *Game) Layout(width, height int) (int, int) {
	return width, height
}

func (g *Ground) Draw(screen *ebiten.Image) {
	// 绘制地面
	for i := 0; i < g.game.width; i += g.grass1.Width {
		for j := 0; j < g.game.height; j += g.grass1.Width {
			grass := g.grass1
			if i%(g.grass1.Width*2) == 0 {
				grass = g.grass2
			}
			options := &ebiten.DrawImageOptions{}
			options.GeoM.Translate(float64(i), float64(j))
			options.ColorScale.SetG(0.9)
			screen.DrawImage(GetSpriteImage(g.game.spriteImages, grass), options)
		}
	}

	for tree := g.trees; tree != nil; tree = tree.Next {
		tree.Value.Draw(screen)
	}
}

func (g *Game) initGround() {
	g.ground = &Ground{
		game:   g,
		grass1: g.spritesInfos["tileGrass1"],
		grass2: g.spritesInfos["tileGrass1"],
	}

	info1 := g.spritesInfos["treeGreen_large"]
	info2 := g.spritesInfos["treeBrown_large"]
	for i := 0; i < len(GroundTrees); i++ {
		info := info1
		if i == 0 || i == 1 || i == 4 {
			info = info2
		}
		g.ground.trees = &Chain[*BoxSprite]{
			Value: &BoxSprite{
				Img: GetSpriteImage(g.spriteImages, info),
				X:   math.Min(float64(g.width)*GroundTrees[i][0], float64(g.width-info.Width)),
				Y:   math.Min(float64(g.height)*GroundTrees[i][1], float64(g.height-info.Height)),
				W:   float64(info.Width),
				H:   float64(info.Height),
			},
			Next: g.ground.trees,
		}
	}
}

func (g *Game) initHero() {
	// 创建玩家
	sprite := g.spritesInfos[TankNames[0]]
	g.hero = &Hero{
		Tank: &Tank{
			BoxSprite: &BoxSprite{
				Img: GetSpriteImage(g.spriteImages, sprite),
				A:   AnglePi,
				X:   float64(g.width-sprite.Height) / 2,
				Y:   float64(g.height-sprite.Height) / 2,
				W:   float64(sprite.Height),
				H:   float64(sprite.Height),
			},
			game:          g,
			typ:           0,
			speed:         TankSpeeds[0],
			bulletSize:    2,
			bulletSpeed:   BulletSpeeds[0],
			shootCoolDown: int(BulletSpeeds[0]),
			hitStatus:     180,
			hitProtect:    180,
			hitSprites:    g.tankHitSprites(),
			life:          9,
			maxLife:       9,
		},
	}
}

func (g *Game) initEnemies() {
	// 创建敌人
	g.enemy = nil
	for i := 0; i < 10; i++ {
		typ := 1 + i%(len(TankNames)-1)
		sprite := g.spritesInfos[TankNames[typ]]
		enemy := &Chain[*Enemy]{
			Value: &Enemy{
				Tank: &Tank{
					BoxSprite: &BoxSprite{
						Img: GetSpriteImage(g.spriteImages, sprite),
						A:   TankAngles[rand.Intn(len(TankAngles))],
						X:   float64(rand.Intn(g.width - sprite.Height)),
						Y:   float64(rand.Intn(g.height - sprite.Height)),
						W:   float64(sprite.Height),
						H:   float64(sprite.Height),
					},
					game:          g,
					typ:           typ,
					maxLife:       typ,
					speed:         TankSpeeds[typ],
					bulletSize:    1.2,
					bulletSpeed:   BulletSpeeds[typ],
					shootCoolDown: int(BulletSpeeds[typ]),
					hitSprites:    g.tankHitSprites(),
				},
			},
			Next: g.enemy,
		}
		enemy.Value.reborn() // 出生
		g.enemy = enemy
	}
}

func (g *Game) tankHitSprites() [5]SpriteInfo {
	return [5]SpriteInfo{
		g.spritesInfos["explosion5"],
		g.spritesInfos["explosion4"],
		g.spritesInfos["explosion3"],
		g.spritesInfos["explosion2"],
		g.spritesInfos["explosion1"]}
}

func newPlayer(reader io.Reader) *audio.Player {
	stream, err := vorbis.DecodeWithoutResampling(reader)
	FatalIfError(err)
	player, err := AudioCtx.NewPlayer(stream)
	FatalIfError(err)
	player.SetVolume(0.2)
	return player
}

func newInfinitePlayer(reader io.Reader) *audio.Player {
	stream, err := vorbis.DecodeWithoutResampling(reader)
	FatalIfError(err)
	infiniteLoop := audio.NewInfiniteLoop(stream, stream.Length())
	player, err := AudioCtx.NewPlayer(infiniteLoop)
	FatalIfError(err)
	player.SetVolume(0.2)
	return player
}

func (g *Game) getIconImage() *ebiten.Image {
	tankInfo := g.spritesInfos["tank_sand"]
	iconImage := ebiten.NewImage(tankInfo.Width, tankInfo.Height)
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Translate(float64(-tankInfo.X-tankInfo.Width), float64(-tankInfo.Y-tankInfo.Height))
	options.GeoM.Rotate(AnglePi)
	iconImage.DrawImage(g.spriteImages, options)
	return iconImage
}

func FatalIfError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
