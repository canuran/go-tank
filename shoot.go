package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"math/rand"
)

type Bullet struct {
	*BoxSprite
	game  *Game
	speed float64
	tank  *Tank
	next  *Bullet
}

func (b *Bullet) AutoMove() {
	if b.A == AngleZero {
		b.Y = b.Y - b.speed
	}
	if b.A == AnglePi {
		b.Y = b.Y + b.speed
	}
	if b.A == AngleTrebleHalfPi {
		b.X = b.X - b.speed
	}
	if b.A == AngleHalfPi {
		b.X = b.X + b.speed
	}
}

func (b *Bullet) HitCheck() {
	// 子弹是否与敌方坦克碰撞
	if b.tank == b.game.hero.Tank {
		for other := b.game.enemy; other != nil; other = other.Next {
			if b.hitTank(other.Value.Tank) || b.hitBullets(other.Value.Tank.bullet) {
				b.game.score += int(other.Value.speed)
				if b.game.highScore < b.game.score {
					b.game.highScore = b.game.score
				}
				return
			}
		}
	} else {
		if b.hitTank(b.game.hero.Tank) || b.hitBullets(b.game.hero.Tank.bullet) {
			return
		}
	}

	b.hitTrees()
}

func (b *Bullet) hitTrees() bool {
	// 子弹是否与树碰撞
	for tree := b.game.ground.trees; tree != nil; tree = tree.Next {
		if cx, cy := b.CollideXY(tree.Value); cx != 0 && cy != 0 {
			b.X = -1000 // 子弹失效
			return true
		}
	}
	return false
}

func (b *Bullet) hitTank(other *Tank) bool {
	// 是否击中敌方坦克
	if cx, cy := b.CollideXY(other.BoxSprite); cx != 0 && cy != 0 {
		if other.life > 0 { // 活着的坦克才能被击中
			if other.hitStatus < 1 { // 坦克未受攻击保护
				other.life--
				if other.life < 1 {
					other.hitStatus = DieHitStatus
					_ = other.game.explodeAudio.Rewind()
					other.game.explodeAudio.Play()
				} else {
					other.hitStatus = other.hitProtect
					_ = other.game.hitAudio.Rewind()
					other.game.hitAudio.Play()
				}
			}
			b.X = -1000 // 子弹失效
			return true
		}
	}
	return false
}

func (b *Bullet) hitBullets(bullet *Bullet) bool {
	// 子弹是否与敌方子弹碰撞
	for ; bullet != nil; bullet = bullet.next {
		if cx, cy := b.CollideXY(bullet.BoxSprite); cx != 0 && cy != 0 {
			b.X = -1000      // 子弹失效
			bullet.X = -1000 // 子弹失效
			return true
		}
	}
	return false
}

func (h *Hero) UpdateShoot() {
	h.UpdateBullet()
	if !h.checkHealth() {
		return
	}
	if h.shootCool < ShootCooled {
		h.shootCool += h.shootCoolDown
	} else if ebiten.IsKeyPressed(ebiten.KeyEnter) || ebiten.IsKeyPressed(ebiten.KeyControl) ||
		ebiten.IsStandardGamepadButtonPressed(GamepadID, ebiten.StandardGamepadButtonFrontBottomLeft) ||
		ebiten.IsStandardGamepadButtonPressed(GamepadID, ebiten.StandardGamepadButtonFrontBottomRight) ||
		ebiten.IsStandardGamepadButtonPressed(GamepadID, ebiten.StandardGamepadButtonRightTop) ||
		ebiten.IsStandardGamepadButtonPressed(GamepadID, ebiten.StandardGamepadButtonRightLeft) ||
		ebiten.IsStandardGamepadButtonPressed(GamepadID, ebiten.StandardGamepadButtonRightRight) ||
		ebiten.IsStandardGamepadButtonPressed(GamepadID, ebiten.StandardGamepadButtonRightBottom) {
		h.shootBullet()
	}
}

func (h *Hero) checkHealth() bool {
	if h.hitStatus > 0 {
		h.hitStatus--
		if h.hitStatus == 0 && h.life < 1 {
			h.game.Restart()
			return false
		}
	}
	return true
}

func (tk *Tank) UpdateBullet() {
	// 分多次移动避免跳过碰撞
	for i := 0; i < 4; i++ {
		var preBullet *Bullet
		for bullet := tk.bullet; bullet != nil; bullet = bullet.next {
			bullet.AutoMove()
			bullet.HitCheck()
			preBullet = tk.removeInvalidBullet(preBullet, bullet)
		}
	}
}

func (e *Enemy) AutoShoot() {
	e.UpdateBullet()
	if !e.checkHealth() {
		return
	}
	if e.shootCool < ShootCooled {
		e.shootCool += e.shootCoolDown
	} else if e.game.updates%(1+rand.Intn(120)) == 0 {
		e.bulletSpeed = BulletSpeeds[e.typ] * (1 + float64(e.game.score)/1000)
		e.shootBullet()
	}
}

func (e *Enemy) checkHealth() bool {
	if e.hitStatus > 0 {
		e.hitStatus--
		if e.hitStatus == 0 && e.life < 1 {
			e.reborn()
			return false
		}
	}
	return true
}

func (e *Enemy) reborn() {
	// 重生在随机位置且不碰撞
	e.life = e.maxLife
	e.shootCool = -180
	minX, minY, maxX, maxY := float64(1), float64(1), float64(1), float64(1)
	for minX != 0 || minY != 0 || maxX != 0 || maxY != 0 {
		e.X = e.W + float64(rand.Intn(e.game.width-int(e.W)*2))
		e.Y = e.H + float64(rand.Intn(e.game.height-int(e.H*2)))
		minX, minY, maxX, maxY = e.CollideOthers()
	}
}

func (tk *Tank) shootBullet() {
	tk.shootCool = 0
	img := GetSpriteImage(tk.game.spriteImages, tk.game.spritesInfos[BulletNames[tk.typ]])
	tk.bullet = &Bullet{
		BoxSprite: &BoxSprite{
			Img: img,
			A:   tk.A,
			X:   tk.X,
			Y:   tk.Y,
			W:   float64(img.Bounds().Dx()) * tk.bulletSize,
			H:   float64(img.Bounds().Dy()) * tk.bulletSize,
		},
		game:  tk.game,
		speed: tk.bulletSpeed / 4,
		tank:  tk,
		next:  tk.bullet,
	}

	// 调整子弹的初始角度和位置
	w, h := tk.GetDrawWH()
	if tk.A == AngleZero {
		tk.bullet.A = AnglePi
		tk.bullet.X += w/2 - tk.bullet.W/2
		tk.bullet.Y += h
	} else if tk.A == AnglePi {
		tk.bullet.A = AngleZero
		tk.bullet.X += w/2 - tk.bullet.W/2
		tk.bullet.Y -= tk.bullet.H
	} else if tk.A == AngleHalfPi {
		tk.bullet.A = AngleTrebleHalfPi
		tk.bullet.X -= tk.bullet.H
		tk.bullet.Y += h/2 - tk.bullet.W/2
	} else if tk.A == AngleTrebleHalfPi {
		tk.bullet.A = AngleHalfPi
		tk.bullet.X += w
		tk.bullet.Y += h/2 - tk.bullet.W/2
	}
}

func (tk *Tank) removeInvalidBullet(preBullet *Bullet, bullet *Bullet) *Bullet {
	if bullet.X < 0 || bullet.Y < 0 ||
		bullet.X > float64(tk.game.width) || bullet.Y > float64(tk.game.height) {
		if preBullet == nil {
			tk.bullet = bullet.next
		} else {
			preBullet.next = bullet.next
		}
		return preBullet
	} else {
		return bullet
	}
}
