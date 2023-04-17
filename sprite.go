package main

import (
	"bytes"
	_ "embed"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"image"
	"image/png"
	_ "image/png"
	"math"
	"os"
	"strconv"
)

const (
	Precision         = 1e-8
	AngleZero         = 0
	AngleHalfPi       = math.Pi / 2
	AnglePi           = math.Pi
	AngleTrebleHalfPi = math.Pi * 3 / 2
)

var (
	//go:embed sprites.png
	spritesData []byte
	whiteImage  = ebiten.NewImage(1, 1)
)

func init() {
	whiteImage.Fill(colornames.White)
}

type BoxSprite struct {
	Img *ebiten.Image
	A   float64
	X   float64
	Y   float64
	W   float64
	H   float64
}

type Chain[T any] struct {
	Value T
	Next  *Chain[T]
}

// Draw 绘制图形
func (s *BoxSprite) Draw(screen *ebiten.Image) {
	options := &ebiten.DrawImageOptions{}
	// 缩放只针对原始图片，所以先缩放
	options.GeoM.Scale(s.W/float64(s.Img.Bounds().Dx()),
		s.H/float64(s.Img.Bounds().Dy()))

	// 先移动到图片中心再旋转
	options.GeoM.Translate(-s.W/2, -s.H/2)
	options.GeoM.Rotate(s.A)

	// 移动到屏幕指定位置并修正坐标
	w, h := s.GetDrawWH()
	options.GeoM.Translate(s.X+w/2, s.Y+h/2)
	screen.DrawImage(s.Img, options)
	// s.DrawBorder(screen)
}

func (s *BoxSprite) GetDrawWH() (float64, float64) {
	sin, cos := math.Sincos(s.A)
	return math.Abs(s.W*cos + s.H*sin), math.Abs(s.W*sin + s.H*cos)
}

// DrawBorder 绘制边框
func (s *BoxSprite) DrawBorder(screen *ebiten.Image) {
	w, h := s.GetDrawWH()
	var path vector.Path
	path.MoveTo(float32(s.X), float32(s.Y))
	path.LineTo(float32(s.X+w), float32(s.Y))
	path.LineTo(float32(s.X+w), float32(s.Y+h))
	path.LineTo(float32(s.X), float32(s.Y+h))
	path.Close()
	ops := &vector.StrokeOptions{}
	ops.Width = 2
	vs, is := path.AppendVerticesAndIndicesForStroke([]ebiten.Vertex{}, []uint16{}, ops)
	options := &ebiten.DrawTrianglesOptions{}
	screen.DrawTriangles(vs, is, whiteImage, options)
}

// CollideXY 注意cx和xy同时不为0才存在碰撞
// cx 小于0则位于碰撞左方，否则在右方
// cy 小于0则位于碰撞上方，否则在下方
func (s *BoxSprite) CollideXY(sp *BoxSprite) (cx, cy float64) {
	w1, h1 := s.GetDrawWH()
	w2, h2 := sp.GetDrawWH()
	// 计算矩形中心的X和Y轴的距离
	dX := s.X + w1/2 - sp.X - w2/2
	if dX < 0 { // 在左方
		cx = math.Min(-dX-(w1/2+w2/2), 0)
	} else { // 在右方
		cx = math.Max((w1/2+w2/2)-dX, 0)
	}
	dY := s.Y + h1/2 - sp.Y - h2/2
	if dY < 0 { // 在上方
		cy = math.Min(-dY-(h1/2+h2/2), 0)
	} else { // 在下方
		cy = math.Max((h1/2+h2/2)-dY, 0)
	}
	// 修正浮点计算的精度
	if cx < Precision && cx > -Precision {
		cx = 0
	}
	if cy < Precision && cy > -Precision {
		cy = 0
	}
	return
}

type SpriteInfo struct {
	Name   string `json:"name,omitempty"`
	X      int    `json:"x,omitempty"`
	Y      int    `json:"y,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

func GetSpriteImage(img *ebiten.Image, info SpriteInfo) *ebiten.Image {
	return img.SubImage(image.Rect(info.X, info.Y,
		info.X+info.Width, info.Y+info.Height)).(*ebiten.Image)
}

func LoadSpritesImage() *ebiten.Image {
	img, _, err := image.Decode(bytes.NewReader(spritesData))
	FatalIfError(err)
	return ebiten.NewImageFromImage(img)
}

func LoadSpriteInfos() map[string]SpriteInfo {
	spriteMap := make(map[string]SpriteInfo, len(allSpriteInfos))
	for _, sprite := range allSpriteInfos {
		spriteMap[sprite.Name] = sprite
	}
	return spriteMap
}

func (g *Game) OutputSpriteInfos() {
	chsFont, err := opentype.Parse(ChsFont)
	FatalIfError(err)
	chsFace, err := opentype.NewFace(chsFont,
		&opentype.FaceOptions{
			Size: 10, DPI: 72,
			Hinting: font.HintingVertical,
		})
	FatalIfError(err)
	scale := 2.5
	img := ebiten.NewImage(int(float64(g.spriteImages.Bounds().Dx())*scale),
		int(float64(g.spriteImages.Bounds().Dy())*scale))
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(scale, scale)
	img.DrawImage(g.spriteImages, options)
	for _, sprite := range g.spritesInfos {
		text.Draw(img, strconv.Itoa(sprite.X), chsFace, int(float64(sprite.X)*scale),
			int(float64(sprite.Y)*scale)+15, colornames.Lightyellow)
		text.Draw(img, strconv.Itoa(sprite.Y), chsFace, int(float64(sprite.X)*scale),
			int(float64(sprite.Y)*scale)+28, colornames.Lightyellow)
		text.Draw(img, strconv.Itoa(sprite.Width), chsFace, int(float64(sprite.X)*scale),
			int(float64(sprite.Y)*scale)+41, colornames.Lightyellow)
		text.Draw(img, strconv.Itoa(sprite.Height), chsFace, int(float64(sprite.X)*scale),
			int(float64(sprite.Y)*scale)+54, colornames.Lightyellow)
	}
	create, err := os.Create("spritex.png")
	FatalIfError(err)
	err = png.Encode(create, img)
	FatalIfError(err)
}

var allSpriteInfos = []SpriteInfo{{
	Name:   "barrelBlack_side",
	X:      1016,
	Y:      510,
	Width:  40,
	Height: 56,
}, {
	Name:   "barrelBlack_top",
	X:      1014,
	Y:      1032,
	Width:  48,
	Height: 48,
}, {
	Name:   "barrelGreen_side",
	X:      1024,
	Y:      0,
	Width:  40,
	Height: 56,
}, {
	Name:   "barrelGreen_top",
	X:      1012,
	Y:      809,
	Width:  48,
	Height: 48,
}, {
	Name:   "barrelRed_side",
	X:      828,
	Y:      740,
	Width:  40,
	Height: 56,
}, {
	Name:   "barrelRed_top",
	X:      1014,
	Y:      984,
	Width:  48,
	Height: 48,
}, {
	Name:   "barrelRust_side",
	X:      1016,
	Y:      753,
	Width:  40,
	Height: 56,
}, {
	Name:   "barrelRust_top",
	X:      1014,
	Y:      936,
	Width:  48,
	Height: 48,
}, {
	Name:   "barricadeMetal",
	X:      958,
	Y:      936,
	Width:  56,
	Height: 56,
}, {
	Name:   "barricadeWood",
	X:      958,
	Y:      1048,
	Width:  56,
	Height: 56,
}, {
	Name:   "bulletBlue1",
	X:      1006,
	Y:      1104,
	Width:  8,
	Height: 20,
}, {
	Name:   "bulletBlue1_outline",
	X:      1106,
	Y:      1069,
	Width:  16,
	Height: 28,
}, {
	Name:   "bulletBlue2",
	X:      990,
	Y:      1104,
	Width:  16,
	Height: 24,
}, {
	Name:   "bulletBlue2_outline",
	X:      1026,
	Y:      705,
	Width:  24,
	Height: 32,
}, {
	Name:   "bulletBlue3",
	X:      870,
	Y:      465,
	Width:  8,
	Height: 28,
}, {
	Name:   "bulletBlue3_outline",
	X:      1107,
	Y:      240,
	Width:  16,
	Height: 36,
}, {
	Name:   "bulletDark1",
	X:      228,
	Y:      1024,
	Width:  8,
	Height: 20,
}, {
	Name:   "bulletDark1_outline",
	X:      1106,
	Y:      1097,
	Width:  16,
	Height: 28,
}, {
	Name:   "bulletDark2",
	X:      974,
	Y:      1104,
	Width:  16,
	Height: 24,
}, {
	Name:   "bulletDark2_outline",
	X:      1085,
	Y:      654,
	Width:  24,
	Height: 32,
}, {
	Name:   "bulletDark3",
	X:      1024,
	Y:      158,
	Width:  8,
	Height: 28,
}, {
	Name:   "bulletDark3_outline",
	X:      1106,
	Y:      492,
	Width:  16,
	Height: 36,
}, {
	Name:   "bulletGreen1",
	X:      308,
	Y:      1104,
	Width:  8,
	Height: 20,
}, {
	Name:   "bulletGreen1_outline",
	X:      1106,
	Y:      412,
	Width:  16,
	Height: 28,
}, {
	Name:   "bulletGreen2",
	X:      684,
	Y:      1099,
	Width:  16,
	Height: 24,
}, {
	Name:   "bulletGreen2_outline",
	X:      1066,
	Y:      1069,
	Width:  24,
	Height: 32,
}, {
	Name:   "bulletGreen3",
	X:      700,
	Y:      1099,
	Width:  8,
	Height: 28,
}, {
	Name:   "bulletGreen3_outline",
	X:      1105,
	Y:      204,
	Width:  16,
	Height: 36,
}, {
	Name:   "bulletRed1",
	X:      236,
	Y:      1024,
	Width:  8,
	Height: 20,
}, {
	Name:   "bulletRed1_outline",
	X:      668,
	Y:      1099,
	Width:  16,
	Height: 28,
}, {
	Name:   "bulletRed2",
	X:      958,
	Y:      1104,
	Width:  16,
	Height: 24,
}, {
	Name:   "bulletRed2_outline",
	X:      1061,
	Y:      654,
	Width:  24,
	Height: 32,
}, {
	Name:   "bulletRed3",
	X:      308,
	Y:      1076,
	Width:  8,
	Height: 28,
}, {
	Name:   "bulletRed3_outline",
	X:      1064,
	Y:      60,
	Width:  16,
	Height: 36,
}, {
	Name:   "bulletSand1",
	X:      212,
	Y:      1108,
	Width:  8,
	Height: 20,
}, {
	Name:   "bulletSand1_outline",
	X:      652,
	Y:      1099,
	Width:  16,
	Height: 28,
}, {
	Name:   "bulletSand2",
	X:      1090,
	Y:      518,
	Width:  16,
	Height: 24,
}, {
	Name:   "bulletSand2_outline",
	X:      1084,
	Y:      120,
	Width:  24,
	Height: 32,
}, {
	Name:   "bulletSand3",
	X:      952,
	Y:      753,
	Width:  8,
	Height: 28,
}, {
	Name:   "bulletSand3_outline",
	X:      930,
	Y:      569,
	Width:  16,
	Height: 36,
}, {
	Name:   "crateMetal",
	X:      958,
	Y:      992,
	Width:  56,
	Height: 56,
}, {
	Name:   "crateMetal_side",
	X:      960,
	Y:      434,
	Width:  56,
	Height: 56,
}, {
	Name:   "crateWood",
	X:      960,
	Y:      753,
	Width:  56,
	Height: 56,
}, {
	Name:   "crateWood_side",
	X:      960,
	Y:      490,
	Width:  56,
	Height: 56,
}, {
	Name:   "explosion1",
	X:      640,
	Y:      804,
	Width:  120,
	Height: 120,
}, {
	Name:   "explosion2",
	X:      764,
	Y:      508,
	Width:  114,
	Height: 112,
}, {
	Name:   "explosion3",
	X:      640,
	Y:      256,
	Width:  127,
	Height: 126,
}, {
	Name:   "explosion4",
	X:      860,
	Y:      96,
	Width:  92,
	Height: 90,
}, {
	Name:   "explosion5",
	X:      0,
	Y:      1024,
	Width:  106,
	Height: 104,
}, {
	Name:   "explosionSmoke1",
	X:      640,
	Y:      924,
	Width:  120,
	Height: 120,
}, {
	Name:   "explosionSmoke2",
	X:      760,
	Y:      940,
	Width:  114,
	Height: 112,
}, {
	Name:   "explosionSmoke3",
	X:      640,
	Y:      382,
	Width:  126,
	Height: 126,
}, {
	Name:   "explosionSmoke4",
	X:      768,
	Y:      96,
	Width:  92,
	Height: 90,
}, {
	Name:   "explosionSmoke5",
	X:      106,
	Y:      1024,
	Width:  106,
	Height: 104,
}, {
	Name:   "fenceRed",
	X:      212,
	Y:      1076,
	Width:  96,
	Height: 32,
}, {
	Name:   "fenceYellow",
	X:      212,
	Y:      1044,
	Width:  104,
	Height: 32,
}, {
	Name:   "oilSpill_large",
	X:      524,
	Y:      1024,
	Width:  100,
	Height: 100,
}, {
	Name:   "oilSpill_small",
	X:      624,
	Y:      1099,
	Width:  28,
	Height: 28,
}, {
	Name:   "sandbagBeige",
	X:      768,
	Y:      186,
	Width:  64,
	Height: 44,
}, {
	Name:   "sandbagBeige_open",
	X:      624,
	Y:      1044,
	Width:  84,
	Height: 55,
}, {
	Name:   "sandbagBrown",
	X:      764,
	Y:      740,
	Width:  64,
	Height: 44,
}, {
	Name:   "sandbagBrown_open",
	X:      708,
	Y:      1052,
	Width:  84,
	Height: 55,
}, {
	Name:   "shotLarge",
	X:      1024,
	Y:      56,
	Width:  40,
	Height: 50,
}, {
	Name:   "shotOrange",
	X:      1033,
	Y:      214,
	Width:  32,
	Height: 56,
}, {
	Name:   "shotRed",
	X:      1016,
	Y:      434,
	Width:  42,
	Height: 76,
}, {
	Name:   "shotThin",
	X:      1106,
	Y:      440,
	Width:  16,
	Height: 52,
}, {
	Name:   "specialBarrel1",
	X:      1014,
	Y:      1080,
	Width:  28,
	Height: 44,
}, {
	Name:   "specialBarrel1_outline",
	X:      1024,
	Y:      106,
	Width:  36,
	Height: 52,
}, {
	Name:   "specialBarrel2",
	X:      1042,
	Y:      1080,
	Width:  24,
	Height: 48,
}, {
	Name:   "specialBarrel2_outline",
	X:      1033,
	Y:      158,
	Width:  32,
	Height: 56,
}, {
	Name:   "specialBarrel3",
	X:      1088,
	Y:      0,
	Width:  20,
	Height: 56,
}, {
	Name:   "specialBarrel3_outline",
	X:      832,
	Y:      186,
	Width:  28,
	Height: 64,
}, {
	Name:   "specialBarrel4",
	X:      1088,
	Y:      746,
	Width:  20,
	Height: 64,
}, {
	Name:   "specialBarrel4_outline",
	X:      1060,
	Y:      765,
	Width:  28,
	Height: 72,
}, {
	Name:   "specialBarrel5",
	X:      1060,
	Y:      106,
	Width:  24,
	Height: 52,
}, {
	Name:   "specialBarrel5_outline",
	X:      1058,
	Y:      362,
	Width:  32,
	Height: 60,
}, {
	Name:   "specialBarrel6",
	X:      1089,
	Y:      152,
	Width:  16,
	Height: 52,
}, {
	Name:   "specialBarrel6_outline",
	X:      1062,
	Y:      897,
	Width:  24,
	Height: 60,
}, {
	Name:   "specialBarrel7",
	X:      1106,
	Y:      360,
	Width:  16,
	Height: 52,
}, {
	Name:   "specialBarrel7_outline",
	X:      1062,
	Y:      1009,
	Width:  24,
	Height: 60,
}, {
	Name:   "tankBlue_barrel1",
	X:      1086,
	Y:      957,
	Width:  24,
	Height: 52,
}, {
	Name:   "tankBlue_barrel1_outline",
	X:      1058,
	Y:      422,
	Width:  32,
	Height: 60,
}, {
	Name:   "tankBlue_barrel2",
	X:      1090,
	Y:      1069,
	Width:  16,
	Height: 52,
}, {
	Name:   "tankBlue_barrel2_outline",
	X:      1060,
	Y:      837,
	Width:  24,
	Height: 60,
}, {
	Name:   "tankBlue_barrel3",
	X:      1090,
	Y:      466,
	Width:  16,
	Height: 52,
}, {
	Name:   "tankBlue_barrel3_outline",
	X:      1061,
	Y:      542,
	Width:  24,
	Height: 60,
}, {
	Name:   "tankBody_bigRed",
	X:      768,
	Y:      0,
	Width:  96,
	Height: 96,
}, {
	Name:   "tankBody_bigRed_outline",
	X:      420,
	Y:      1024,
	Width:  104,
	Height: 104,
}, {
	Name:   "tankBody_blue",
	X:      792,
	Y:      1052,
	Width:  76,
	Height: 76,
}, {
	Name:   "tankBody_blue_outline",
	X:      868,
	Y:      620,
	Width:  84,
	Height: 84,
}, {
	Name:   "tankBody_dark",
	X:      876,
	Y:      864,
	Width:  76,
	Height: 72,
}, {
	Name:   "tankBody_darkLarge",
	X:      767,
	Y:      256,
	Width:  96,
	Height: 112,
}, {
	Name:   "tankBody_darkLarge_outline",
	X:      764,
	Y:      620,
	Width:  104,
	Height: 120,
}, {
	Name:   "tankBody_dark_outline",
	X:      868,
	Y:      704,
	Width:  84,
	Height: 80,
}, {
	Name:   "tankBody_green",
	X:      947,
	Y:      290,
	Width:  76,
	Height: 72,
}, {
	Name:   "tankBody_green_outline",
	X:      874,
	Y:      1032,
	Width:  84,
	Height: 80,
}, {
	Name:   "tankBody_huge",
	X:      760,
	Y:      804,
	Width:  116,
	Height: 136,
}, {
	Name:   "tankBody_huge_outline",
	X:      640,
	Y:      660,
	Width:  124,
	Height: 144,
}, {
	Name:   "tankBody_red",
	X:      1023,
	Y:      290,
	Width:  68,
	Height: 72,
}, {
	Name:   "tankBody_red_outline",
	X:      952,
	Y:      569,
	Width:  76,
	Height: 80,
}, {
	Name:   "tankBody_sand",
	X:      952,
	Y:      864,
	Width:  76,
	Height: 72,
}, {
	Name:   "tankBody_sand_outline",
	X:      876,
	Y:      784,
	Width:  84,
	Height: 80,
}, {
	Name:   "tankDark_barrel1",
	X:      1085,
	Y:      602,
	Width:  24,
	Height: 52,
}, {
	Name:   "tankDark_barrel1_outline",
	X:      1056,
	Y:      705,
	Width:  32,
	Height: 60,
}, {
	Name:   "tankDark_barrel2",
	X:      1091,
	Y:      308,
	Width:  16,
	Height: 52,
}, {
	Name:   "tankDark_barrel2_outline",
	X:      1084,
	Y:      837,
	Width:  24,
	Height: 60,
}, {
	Name:   "tankDark_barrel3",
	X:      1107,
	Y:      276,
	Width:  16,
	Height: 52,
}, {
	Name:   "tankDark_barrel3_outline",
	X:      1065,
	Y:      210,
	Width:  24,
	Height: 60,
}, {
	Name:   "tankGreen_barrel1",
	X:      1062,
	Y:      957,
	Width:  24,
	Height: 52,
}, {
	Name:   "tankGreen_barrel1_outline",
	X:      1028,
	Y:      857,
	Width:  32,
	Height: 60,
}, {
	Name:   "tankGreen_barrel2",
	X:      1108,
	Y:      746,
	Width:  16,
	Height: 52,
}, {
	Name:   "tankGreen_barrel2_outline",
	X:      1086,
	Y:      897,
	Width:  24,
	Height: 60,
}, {
	Name:   "tankGreen_barrel3",
	X:      1089,
	Y:      204,
	Width:  16,
	Height: 52,
}, {
	Name:   "tankGreen_barrel3_outline",
	X:      1086,
	Y:      1009,
	Width:  24,
	Height: 60,
}, {
	Name:   "tankRed_barrel1",
	X:      1061,
	Y:      602,
	Width:  24,
	Height: 52,
}, {
	Name:   "tankRed_barrel1_outline",
	X:      1026,
	Y:      362,
	Width:  32,
	Height: 60,
}, {
	Name:   "tankRed_barrel2",
	X:      1090,
	Y:      414,
	Width:  16,
	Height: 52,
}, {
	Name:   "tankRed_barrel2_outline",
	X:      1085,
	Y:      542,
	Width:  24,
	Height: 60,
}, {
	Name:   "tankRed_barrel3",
	X:      1090,
	Y:      362,
	Width:  16,
	Height: 52,
}, {
	Name:   "tankRed_barrel3_outline",
	X:      1088,
	Y:      686,
	Width:  24,
	Height: 60,
}, {
	Name:   "tankSand_barrel1",
	X:      1065,
	Y:      158,
	Width:  24,
	Height: 52,
}, {
	Name:   "tankSand_barrel1_outline",
	X:      1058,
	Y:      482,
	Width:  32,
	Height: 60,
}, {
	Name:   "tankSand_barrel2",
	X:      1105,
	Y:      152,
	Width:  16,
	Height: 52,
}, {
	Name:   "tankSand_barrel2_outline",
	X:      1064,
	Y:      0,
	Width:  24,
	Height: 60,
}, {
	Name:   "tankSand_barrel3",
	X:      1091,
	Y:      256,
	Width:  16,
	Height: 52,
}, {
	Name:   "tankSand_barrel3_outline",
	X:      1084,
	Y:      60,
	Width:  24,
	Height: 60,
}, {
	Name:   "tank_bigRed",
	X:      316,
	Y:      1024,
	Width:  104,
	Height: 104,
}, {
	Name:   "tank_blue",
	X:      874,
	Y:      940,
	Width:  84,
	Height: 92,
}, {
	Name:   "tank_dark",
	X:      870,
	Y:      373,
	Width:  84,
	Height: 92,
}, {
	Name:   "tank_darkLarge",
	X:      766,
	Y:      382,
	Width:  104,
	Height: 120,
}, {
	Name:   "tank_green",
	X:      864,
	Y:      0,
	Width:  84,
	Height: 92,
}, {
	Name:   "tank_huge",
	X:      640,
	Y:      508,
	Width:  124,
	Height: 152,
}, {
	Name:   "tank_red",
	X:      948,
	Y:      0,
	Width:  76,
	Height: 92,
}, {
	Name:   "tank_sand",
	X:      863,
	Y:      281,
	Width:  84,
	Height: 92,
}, {
	Name:   "tileGrass1",
	X:      384,
	Y:      896,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass2",
	X:      384,
	Y:      256,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadCornerLL",
	X:      0,
	Y:      512,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadCornerLR",
	X:      0,
	Y:      640,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadCornerUL",
	X:      128,
	Y:      256,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadCornerUR",
	X:      128,
	Y:      384,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadCrossing",
	X:      128,
	Y:      640,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadCrossingRound",
	X:      384,
	Y:      512,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadEast",
	X:      0,
	Y:      768,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadNorth",
	X:      128,
	Y:      896,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadSplitE",
	X:      128,
	Y:      768,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadSplitN",
	X:      384,
	Y:      384,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadSplitS",
	X:      384,
	Y:      768,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadSplitW",
	X:      512,
	Y:      512,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadTransitionE",
	X:      512,
	Y:      640,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadTransitionE_dirt",
	X:      512,
	Y:      768,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadTransitionN",
	X:      512,
	Y:      384,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadTransitionN_dirt",
	X:      640,
	Y:      0,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadTransitionS",
	X:      512,
	Y:      896,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadTransitionS_dirt",
	X:      0,
	Y:      0,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadTransitionW",
	X:      0,
	Y:      256,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_roadTransitionW_dirt",
	X:      0,
	Y:      128,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_transitionE",
	X:      640,
	Y:      128,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_transitionN",
	X:      512,
	Y:      256,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_transitionS",
	X:      512,
	Y:      128,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileGrass_transitionW",
	X:      512,
	Y:      0,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileSand1",
	X:      256,
	Y:      128,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileSand2",
	X:      256,
	Y:      0,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileSand_roadCornerLL",
	X:      384,
	Y:      640,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileSand_roadCornerLR",
	X:      0,
	Y:      896,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileSand_roadCornerUL",
	X:      128,
	Y:      128,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileSand_roadCornerUR",
	X:      0,
	Y:      384,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileSand_roadCrossing",
	X:      384,
	Y:      128,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileSand_roadCrossingRound",
	X:      384,
	Y:      0,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileSand_roadEast",
	X:      256,
	Y:      896,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileSand_roadNorth",
	X:      256,
	Y:      768,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileSand_roadSplitE",
	X:      256,
	Y:      640,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileSand_roadSplitN",
	X:      256,
	Y:      512,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileSand_roadSplitS",
	X:      256,
	Y:      384,
	Width:  128,
	Height: 128,
}, {
	Name:   "tileSand_roadSplitW",
	X:      256,
	Y:      256,
	Width:  128,
	Height: 128,
}, {
	Name:   "tracksDouble",
	X:      951,
	Y:      186,
	Width:  82,
	Height: 104,
}, {
	Name:   "tracksLarge",
	X:      878,
	Y:      465,
	Width:  82,
	Height: 104,
}, {
	Name:   "tracksSmall",
	X:      952,
	Y:      649,
	Width:  74,
	Height: 104,
}, {
	Name:   "treeBrown_large",
	X:      128,
	Y:      512,
	Width:  128,
	Height: 128,
}, {
	Name:   "treeBrown_leaf",
	X:      212,
	Y:      1024,
	Width:  16,
	Height: 20,
}, {
	Name:   "treeBrown_small",
	X:      952,
	Y:      92,
	Width:  72,
	Height: 72,
}, {
	Name:   "treeBrown_twigs",
	X:      878,
	Y:      569,
	Width:  52,
	Height: 44,
}, {
	Name:   "treeGreen_large",
	X:      128,
	Y:      0,
	Width:  128,
	Height: 128,
}, {
	Name:   "treeGreen_leaf",
	X:      624,
	Y:      1024,
	Width:  16,
	Height: 20,
}, {
	Name:   "treeGreen_small",
	X:      954,
	Y:      362,
	Width:  72,
	Height: 72,
}, {
	Name:   "treeGreen_twigs",
	X:      960,
	Y:      809,
	Width:  52,
	Height: 44,
}, {
	Name:   "wireCrooked",
	X:      863,
	Y:      186,
	Width:  88,
	Height: 95,
}, {
	Name:   "wireStraight",
	X:      1028,
	Y:      566,
	Width:  33,
	Height: 139,
}}
