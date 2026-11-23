//go:build ignore

package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// Generates a 512x512 high-DPI Windows 11 Fluent Design 2 brand icon for 文档清匣 (DocPurger):
// Luminous Fluent Azure rounded folder-box backdrop + crisp layered white document page with
// warm amber keyword highlight + frosted precision loupe badge.
func main() {
	const size = 512
	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			// Normalize coordinates to [0, 256] design space for crisp 2x supersampled precision
			fx := (float64(x) + 0.5) * 0.5
			fy := (float64(y) + 0.5) * 0.5

			// 1. Outer Fluent squircle base (deep-to-bright Fluent Azure glass)
			dBg := sdRoundedBox(fx-128, fy-128, 114, 114, 52)
			bgAlpha := smoothAlpha(-dBg)
			if bgAlpha <= 0 {
				continue
			}

			// Multi-stop Fluent Blue gradient (#35A7FF top-left -> #0078D4 mid -> #004E96 bottom-right)
			diag := clamp01((fx*0.45 + fy*0.55) / 256.0)
			r, g, b := fluentGradient(diag)
			col := color.NRGBA{R: r, G: g, B: b, A: uint8(bgAlpha * 255)}

			// Soft top-left luminous glow inside squircle
			glowDist := math.Hypot((fx-72)*0.85, (fy-52)*1.1)
			if glow := clamp01(1.0 - glowDist/140.0); glow > 0 {
				col = blend(col, color.NRGBA{190, 238, 255, uint8(glow * glow * 78)})
			}

			// Specular 1.5px glass rim
			rim := math.Abs(dBg+1.4) - 1.0
			if a := smoothAlpha(-rim); a > 0 {
				topFade := clamp01(1.15 - fy/190.0)
				col = blend(col, color.NRGBA{255, 255, 255, uint8(a * (28 + 92*topFade))})
			}

			// 2. Back stacked document card (frosted translucent glass sheet, subtly offset)
			dBack := sdRoundedBox(fx-132, fy-114, 48, 60, 13)
			if a := smoothAlpha(-dBack); a > 0 {
				col = blend(col, color.NRGBA{224, 242, 255, uint8(a * 110)})
			}

			// 3. Soft drop shadow beneath main front document
			dFrontShadow := sdRoundedBox(fx-116, fy-132, 50, 65, 14)
			if dFrontShadow < 18 {
				sh := clamp01(1.0 - math.Max(dFrontShadow, 0)/18.0)
				col = blend(col, color.NRGBA{0, 38, 84, uint8(sh * sh * 72)})
			}

			// 4. Main crisp pearl-white document slab
			dDoc := sdRoundedBox(fx-114, fy-126, 50, 65, 14)
			if a := smoothAlpha(-dDoc); a > 0 {
				tDoc := clamp01((fy - 60) / 132.0)
				dr := lerpU8(255, 240, tDoc)
				dg := lerpU8(255, 246, tDoc)
				db := lerpU8(255, 255, tDoc)
				col = blend(col, color.NRGBA{dr, dg, db, uint8(a * 252)})
			}

			// 5. Document content lines + prominent warm amber keyword highlight bar
			lines := []struct {
				cx, cy, hw, hh float64
				c              color.NRGBA
			}{
				// Top header bar (Fluent Blue)
				{104, 92, 24, 4.5, color.NRGBA{0, 120, 212, 215}},
				// Line 2 left segment (slate)
				{93, 114, 13, 4.2, color.NRGBA{148, 163, 184, 165}},
				// Line 2 highlighted keyword pill (Warm Amber Gold!)
				{127, 114, 16, 6.2, color.NRGBA{251, 191, 36, 245}},
				// Line 3 (soft cool gray)
				{108, 136, 28, 4.2, color.NRGBA{148, 163, 184, 145}},
				// Line 4 (short)
				{99, 156, 19, 4.2, color.NRGBA{148, 163, 184, 125}},
			}
			for _, ln := range lines {
				dLine := sdRoundedBox(fx-ln.cx, fy-ln.cy, ln.hw, ln.hh, ln.hh)
				if a := smoothAlpha(-dLine); a > 0 {
					c := ln.c
					c.A = uint8(float64(c.A) * a)
					col = blend(col, c)
				}
			}

			// 6. Floating frosted glass circular loupe badge (bottom-right)
			bx, by := 168.0, 166.0
			distBadge := math.Hypot(fx-bx, fy-by)

			// Badge shadow
			distBadgeShadow := math.Hypot(fx-bx, fy-(by+5))
			if distBadgeShadow < 48 {
				sh := clamp01(1.0 - math.Max(distBadgeShadow-34, 0)/14.0)
				col = blend(col, color.NRGBA{0, 32, 74, uint8(sh * sh * 95)})
			}

			// Badge deep ocean acrylic disc
			if a := smoothAlpha(38.0 - distBadge); a > 0 {
				tb := clamp01((fy - (by - 38)) / 76.0)
				br := lerpU8(0x18, 0x0A, tb)
				bg := lerpU8(0x5A, 0x2E, tb)
				bb := lerpU8(0xB4, 0x66, tb)
				col = blend(col, color.NRGBA{br, bg, bb, uint8(a * 244)})
			}
			// Luminous white-cyan border ring
			ringDist := math.Abs(distBadge-36.0) - 2.0
			if a := smoothAlpha(-ringDist); a > 0 {
				col = blend(col, color.NRGBA{210, 242, 255, uint8(a * 210)})
			}

			// Magnifier glass circle (crisp pure white)
			lx, ly := 163.5, 161.5
			lensDist := math.Abs(math.Hypot(fx-lx, fy-ly)-12.5) - 2.9
			if a := smoothAlpha(-lensDist); a > 0 {
				col = blend(col, color.NRGBA{255, 255, 255, uint8(a * 255)})
			}
			// Soft glass fill inside lens
			if a := smoothAlpha(10.0 - math.Hypot(fx-lx, fy-ly)); a > 0 {
				col = blend(col, color.NRGBA{125, 211, 252, uint8(a * 72)})
			}
			// Magnifier handle (bright ice cyan)
			handleDist := sdSegment(fx, fy, 173.5, 171.5, 184.5, 182.5) - 3.3
			if a := smoothAlpha(-handleDist); a > 0 {
				col = blend(col, color.NRGBA{125, 211, 252, uint8(a * 255)})
			}

			// Clean sparkle accent (symbolizing "清 / purify")
			sp := sdSparkle(fx-184, fy-74, 15.0)
			if a := smoothAlpha(-sp); a > 0 {
				col = blend(col, color.NRGBA{255, 255, 255, uint8(a * 230)})
			}

			img.SetNRGBA(x, y, col)
		}
	}

	_ = os.MkdirAll("build", 0o755)
	f, err := os.Create("build/appicon.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func fluentGradient(t float64) (uint8, uint8, uint8) {
	// #38BDF8 (top-left airy azure) -> #0078D4 (Fluent Blue) -> #00529B (deep sapphire)
	if t < 0.52 {
		u := t / 0.52
		return lerpU8(0x36, 0x00, u), lerpU8(0xB4, 0x78, u), lerpU8(0xF8, 0xD4, u)
	}
	u := (t - 0.52) / 0.48
	return lerpU8(0x00, 0x00, u), lerpU8(0x78, 0x4E, u), lerpU8(0xD4, 0x98, u)
}

func sdRoundedBox(px, py, bx, by, r float64) float64 {
	qx := math.Abs(px) - bx + r
	qy := math.Abs(py) - by + r
	return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
}

func sdSegment(px, py, ax, ay, bx, by float64) float64 {
	pax, pay := px-ax, py-ay
	bax, bay := bx-ax, by-ay
	h := clamp01((pax*bax + pay*bay) / (bax*bax + bay*bay))
	return math.Hypot(pax-bax*h, pay-bay*h)
}

// 4-point crisp sparkle SDF
func sdSparkle(px, py, r float64) float64 {
	ax, ay := math.Abs(px), math.Abs(py)
	// Superellipse-like star approximation
	v := math.Pow(ax/r, 0.58) + math.Pow(ay/r, 0.58)
	return (v - 1.0) * r * 0.55
}

func smoothAlpha(d float64) float64 { return clamp01(d + 0.5) }
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
func lerpU8(a, b uint8, t float64) uint8 { return uint8(float64(a) + (float64(b)-float64(a))*t) }

func blend(dst, src color.NRGBA) color.NRGBA {
	sa := float64(src.A) / 255.0
	da := float64(dst.A) / 255.0
	outA := sa + da*(1-sa)
	if outA <= 0 {
		return color.NRGBA{}
	}
	r := (float64(src.R)*sa + float64(dst.R)*da*(1-sa)) / outA
	g := (float64(src.G)*sa + float64(dst.G)*da*(1-sa)) / outA
	b := (float64(src.B)*sa + float64(dst.B)*da*(1-sa)) / outA
	return color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(outA * 255)}
}
