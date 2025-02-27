package giuwidget

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"time"

	"github.com/ianling/giu"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2datautils"

	"github.com/OpenDiablo2/HellSpawner/hscommon/hsutil"
)

const miliseconds = 1000

type animationPlayMode byte

const (
	playModeForward animationPlayMode = iota
	playModeBackword
	playModePingPong
)

func (a animationPlayMode) String() string {
	s := map[animationPlayMode]string{
		playModeForward:  "Forwards",
		playModeBackword: "Backwords",
		playModePingPong: "Ping-Pong",
	}

	k, ok := s[a]
	if !ok {
		return "Unknown"
	}

	return k
}

const defaultTickTime = 100

type widgetState struct {
	controls struct {
		direction int32
		frame     int32
		scale     int32
	}

	isPlaying bool
	repeat    bool
	tickTime  int32
	playMode  animationPlayMode

	// cache - will not be saved
	images   []*image.RGBA
	textures []*giu.Texture

	isForward bool // determines a direction of animation
	ticker    *time.Ticker
}

// Dispose cleans viewers state
func (s *widgetState) Dispose() {
	s.textures = nil
}

func (s *widgetState) Encode() []byte {
	sw := d2datautils.CreateStreamWriter()

	sw.PushInt32(s.controls.direction)
	sw.PushInt32(s.controls.frame)
	sw.PushInt32(s.controls.scale)

	sw.PushBytes(byte(hsutil.BoolToInt(s.isPlaying)))
	sw.PushBytes(byte(hsutil.BoolToInt(s.repeat)))

	sw.PushInt32(s.tickTime)
	sw.PushBytes(byte(s.playMode))

	return sw.GetBytes()
}

func (s *widgetState) Decode(data []byte) {
	var err error

	sr := d2datautils.CreateStreamReader(data)

	s.controls.direction, err = sr.ReadInt32()
	if err != nil {
		log.Print(err)

		return
	}

	s.controls.frame, err = sr.ReadInt32()
	if err != nil {
		log.Print(err)

		return
	}

	s.controls.scale, err = sr.ReadInt32()
	if err != nil {
		log.Print(err)

		return
	}

	isPlaying, err := sr.ReadByte()
	if err != nil {
		log.Print(err)
		return
	}

	s.isPlaying = (isPlaying == 1)

	repeat, err := sr.ReadByte()
	if err != nil {
		log.Print(err)
		return
	}

	s.repeat = (repeat == 1)

	s.tickTime, err = sr.ReadInt32()
	if err != nil {
		log.Print(err)
		return
	}

	playMode, err := sr.ReadByte()
	if err != nil {
		log.Print(err)
		return
	}

	s.playMode = animationPlayMode(playMode)

	// update ticker
	s.ticker.Reset(time.Second * time.Duration(s.tickTime) / miliseconds)
}

func (p *widget) getStateID() string {
	return fmt.Sprintf("widget_%s", p.id)
}

func (p *widget) getState() *widgetState {
	var state *widgetState

	s := giu.Context.GetState(p.getStateID())

	if s != nil {
		state = s.(*widgetState)
	} else {
		p.initState()
		state = p.getState()
	}

	return state
}

func (p *widget) initState() {
	// Prevent multiple invocation to LoadImage.
	state := &widgetState{
		isPlaying: false,
		repeat:    false,
		tickTime:  defaultTickTime,
		playMode:  playModeForward,
	}

	state.ticker = time.NewTicker(time.Second * time.Duration(state.tickTime) / miliseconds)

	p.setState(state)

	go p.runPlayer(state)

	numDirections := len(p.dcc.Directions())
	numFrames := len(p.dcc.Direction(0).Frames())
	totalFrames := numDirections * numFrames
	state.images = make([]*image.RGBA, totalFrames)

	directions := p.dcc.Directions()
	for dirIdx, direction := range directions {
		fw := p.dcc.Direction(dirIdx).Box.Dx()
		fh := p.dcc.Direction(dirIdx).Box.Dy()

		frames := direction.Frames()
		for frameIdx := range frames {
			absoluteFrameIdx := (dirIdx * numFrames) + frameIdx

			frame := directions[dirIdx].Frame(frameIdx)
			pixels := frame.PixelData

			state.images[absoluteFrameIdx] = image.NewRGBA(image.Rect(0, 0, fw, fh))

			for y := 0; y < fh; y++ {
				for x := 0; x < fw; x++ {
					idx := x + (y * fw)
					if idx >= len(pixels) {
						continue
					}

					RGBAColor := p.makeImagePixel(uint32(pixels[idx]))
					state.images[absoluteFrameIdx].Set(x, y, RGBAColor)
				}
			}
		}
	}

	go func() {
		textures := make([]*giu.Texture, totalFrames)

		for frameIndex := 0; frameIndex < totalFrames; frameIndex++ {
			frameIndex := frameIndex
			p.textureLoader.CreateTextureFromARGB(state.images[frameIndex], func(t *giu.Texture) {
				textures[frameIndex] = t
			})
		}

		s := p.getState()
		s.textures = textures
		p.setState(s)
	}()
}

func (p *widget) setState(s giu.Disposable) {
	giu.Context.SetState(p.getStateID(), s)
}

func (p *widget) makeImagePixel(val uint32) color.RGBA {
	alpha := maxAlpha

	if val == 0 {
		alpha = 0
	}

	var r, g, b uint32

	palette := p.dcc.Palette()
	if palette != nil {
		col := palette[val]
		r, g, b, _ = col.RGBA()
	} else {
		r, g, b = val, val, val
	}

	RGBAColor := color.RGBA{
		R: uint8(r),
		G: uint8(g),
		B: uint8(b),
		A: alpha,
	}

	return RGBAColor
}

func (p *widget) runPlayer(state *widgetState) {
	for range state.ticker.C {
		if !state.isPlaying {
			continue
		}

		numFrames := len(p.dcc.Direction(0).Frames())
		isLastFrame := state.controls.frame == int32(numFrames - 1)

		// update play direction
		switch state.playMode {
		case playModeForward:
			state.isForward = true
		case playModeBackword:
			state.isForward = false
		case playModePingPong:
			if isLastFrame || state.controls.frame == 0 {
				state.isForward = !state.isForward
			}
		}

		// now update the frame number
		if state.isForward {
			state.controls.frame++
		} else {
			state.controls.frame--
		}

		state.controls.frame = int32(hsutil.Wrap(int(state.controls.frame), numFrames))

		// next, check for stopping/repeat
		isStoppingFrame := (state.controls.frame == 0) || (state.controls.frame == int32(numFrames-1))

		if isStoppingFrame && !state.repeat {
			state.isPlaying = false
		}
	}
}
