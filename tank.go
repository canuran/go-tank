package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"math"
	"math/rand"
	"strconv"
)

const (
	ShootCooled  = 180
	DieHitStatus = 30
)

var (
	keyUpUpdates    int64
	keyDownUpdates  int64
	keyLeftUpdates  int64
	keyRightUpdates int64
)

type Tank struct {
	*BoxSprite
	typ           int
	game          *Game
	life          int
	maxLife       int
	speed         float64
	bulletSize    float64
	bulletSpeed   float64
	bullet        *Bullet
	shootCool     int           // 射击冷却程度
	shootCoolDown int           // 射击冷却速度
	hitSprites    [5]SpriteInfo // 爆炸动画，被击中且死亡时播放
	hitStatus     int           // 大于0表示被击中，免疫攻击
	hitProtect    int           // 击中后的免疫时间
}

type Hero struct {
	*Tank
}

type Enemy struct {
	*Tank
}

func (tk *Tank) Draw(screen *ebiten.Image) {
	if tk.hitStatus > 0 {
		if tk.life > 0 {
			tk.BoxSprite.Draw(screen)
			tk.BoxSprite.DrawBorder(screen)
		} else {
			options := &ebiten.DrawImageOptions{}
			options.GeoM.Translate(tk.X, tk.Y)
			sprite := tk.hitSprites[(tk.hitStatus*len(tk.hitSprites)-1)/DieHitStatus]
			screen.DrawImage(GetSpriteImage(tk.game.spriteImages, sprite), options)
		}
	} else {
		tk.BoxSprite.Draw(screen)
	}
	if tk.life > 0 {
		text.Draw(screen, strconv.Itoa(tk.life), tk.game.chsFont,
			int(tk.X+float64(tk.W)/2-5), int(tk.Y+float64(tk.H)/2+5),
			LifeColors[(tk.life*len(LifeColors)-1)/tk.maxLife])
	}
	bullet := tk.bullet
	for bullet != nil {
		bullet.Draw(screen)
		bullet = bullet.next
	}
}

func (tk *Tank) CollideOthers() (minX, minY, maxX, maxY float64) {
	// 与其他坦克的碰撞检测
	for other := tk.game.enemy; other != nil; other = other.Next {
		if tk != other.Value.Tank && other.Value.life > 0 {
			if cx, cy := tk.CollideXY(other.Value.BoxSprite); cx != 0 && cy != 0 {
				maxX = math.Max(cx, maxX)
				minX = math.Min(cx, minX)
				maxY = math.Max(cy, maxY)
				minY = math.Min(cy, minY)
			}
		}
	}

	// 敌人需要判断与英雄的碰撞
	if tk != tk.game.hero.Tank {
		if cx, cy := tk.CollideXY(tk.game.hero.Tank.BoxSprite); cx != 0 && cy != 0 {
			maxX = math.Max(cx, maxX)
			minX = math.Min(cx, minX)
			maxY = math.Max(cy, maxY)
			minY = math.Min(cy, minY)
		}
	}

	// 坦克与树的碰撞检测
	for tree := tk.game.ground.trees; tree != nil; tree = tree.Next {
		if cx, cy := tk.CollideXY(tree.Value); cx != 0 && cy != 0 {
			maxX = math.Max(cx, maxX)
			minX = math.Min(cx, minX)
			maxY = math.Max(cy, maxY)
			minY = math.Min(cy, minY)
		}
	}
	return
}

func (h *Hero) UpdateMove() {
	if h.life < 1 {
		return
	}
	minKeyUpdates := getMinKeyUpdates()
	if minKeyUpdates < math.MaxInt64 {
		// 控制坦克方向
		if minKeyUpdates == keyUpUpdates {
			h.A = AnglePi
			h.Tank.Move()
		}
		if minKeyUpdates == keyDownUpdates {
			h.A = AngleZero
			h.Tank.Move()
		}
		if minKeyUpdates == keyLeftUpdates {
			h.A = AngleHalfPi
			h.Tank.Move()
		}
		if minKeyUpdates == keyRightUpdates {
			h.A = AngleTrebleHalfPi
			h.Tank.Move()
		}
	}
}

func getMinKeyUpdates() int64 {
	var minKeyUpdates int64 = math.MaxInt64
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) ||
		ebiten.IsStandardGamepadButtonPressed(GamepadID, ebiten.StandardGamepadButtonLeftTop) ||
		ebiten.StandardGamepadAxisValue(GamepadID, ebiten.StandardGamepadAxisLeftStickVertical) < -0.4 ||
		ebiten.StandardGamepadAxisValue(GamepadID, ebiten.StandardGamepadAxisRightStickVertical) < -0.4 {
		keyUpUpdates++
		minKeyUpdates = minInt64(keyUpUpdates, minKeyUpdates)
	} else {
		keyUpUpdates = 0
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) ||
		ebiten.IsStandardGamepadButtonPressed(GamepadID, ebiten.StandardGamepadButtonLeftBottom) ||
		ebiten.StandardGamepadAxisValue(GamepadID, ebiten.StandardGamepadAxisLeftStickVertical) > 0.4 ||
		ebiten.StandardGamepadAxisValue(GamepadID, ebiten.StandardGamepadAxisRightStickVertical) > 0.4 {
		keyDownUpdates++
		minKeyUpdates = minInt64(keyDownUpdates, minKeyUpdates)
	} else {
		keyDownUpdates = 0
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) ||
		ebiten.IsStandardGamepadButtonPressed(GamepadID, ebiten.StandardGamepadButtonLeftLeft) ||
		ebiten.StandardGamepadAxisValue(GamepadID, ebiten.StandardGamepadAxisLeftStickHorizontal) < -0.4 ||
		ebiten.StandardGamepadAxisValue(GamepadID, ebiten.StandardGamepadAxisRightStickHorizontal) < -0.4 {
		keyLeftUpdates++
		minKeyUpdates = minInt64(keyLeftUpdates, minKeyUpdates)
	} else {
		keyLeftUpdates = 0
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) ||
		ebiten.IsStandardGamepadButtonPressed(GamepadID, ebiten.StandardGamepadButtonLeftRight) ||
		ebiten.StandardGamepadAxisValue(GamepadID, ebiten.StandardGamepadAxisLeftStickHorizontal) > 0.4 ||
		ebiten.StandardGamepadAxisValue(GamepadID, ebiten.StandardGamepadAxisRightStickHorizontal) > 0.4 {
		keyRightUpdates++
		minKeyUpdates = minInt64(keyRightUpdates, minKeyUpdates)
	} else {
		keyRightUpdates = 0
	}

	return minKeyUpdates
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func (e *Enemy) AutoMove() {
	if e.life < 1 {
		return
	}
	if e.game.updates%(1+rand.Intn(180)) == 0 {
		e.A = TankAngles[rand.Intn(len(TankAngles))]
	}
	e.speed = TankSpeeds[e.typ] * (1 + float64(e.game.score)/1000)
	e.Tank.Move()
}

func (tk *Tank) Move() {
	if tk.A == AnglePi {
		tk.Y = tk.Y - tk.speed
	}
	if tk.A == AngleZero {
		tk.Y = tk.Y + tk.speed
	}
	if tk.A == AngleHalfPi {
		tk.X = tk.X - tk.speed
	}
	if tk.A == AngleTrebleHalfPi {
		tk.X = tk.X + tk.speed
	}
	minX, minY, maxX, maxY := tk.CollideOthers()
	if tk.A == AnglePi || tk.A == AngleZero {
		tk.Y = tk.Y + minY + maxY
	}
	if tk.A == AngleHalfPi || tk.A == AngleTrebleHalfPi {
		tk.X = tk.X + minX + maxX
	}
	// 限制不能超出屏幕
	dw, dh := tk.GetDrawWH()
	tk.X = math.Max(0, math.Min(tk.X, float64(tk.game.width)-dw))
	tk.Y = math.Max(0, math.Min(tk.Y, float64(tk.game.height)-dh))
}
