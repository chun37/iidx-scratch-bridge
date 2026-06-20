// scratch-bridge converts the analog scratch axis of an IIDX entry
// model into clean, 133-ms keyboard pulses that mimic a stock PS2
// controller, so PCSX2 + PS2 IIDX 14 GOLD stops emitting phantom POOR
// judgments on a single wheel motion.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/chun37/iidx-scratch-bridge/internal/config"
	hidpkg "github.com/chun37/iidx-scratch-bridge/internal/hid"
	"github.com/chun37/iidx-scratch-bridge/internal/keyout"
	"github.com/chun37/iidx-scratch-bridge/internal/pulse"
	"github.com/chun37/iidx-scratch-bridge/internal/rotation"
)

func main() {
	configPath := flag.String("config", "config.toml", "path to config TOML")
	dump := flag.Bool("dump", false, "log every Input Report instead of sending keys (probe mode)")
	flag.Parse()

	if !filepath.IsAbs(*configPath) {
		if cwd, err := os.Getwd(); err == nil {
			*configPath = filepath.Join(cwd, *configPath)
		}
	}

	logger := log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)
	logger.Printf("scratch-bridge starting; config=%s", *configPath)

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatalf("config: %v", err)
	}

	sender := keyout.NewWinSender()

	pulser := pulse.New(
		time.Duration(cfg.PulseMs)*time.Millisecond,
		uint16(cfg.UpVK),
		uint16(cfg.DownVK),
		func(vk uint16, down bool) {
			if err := sender.Send(vk, down); err != nil {
				logger.Printf("sendinput: %v", err)
			}
		},
	)
	defer pulser.Close()

	detector := rotation.New(cfg.Threshold)
	prevButtons := atomic.Uint32{}
	// Initialise to a sentinel so the very first report cannot spuriously
	// look like "every button just changed". We trust the first report
	// itself as the baseline.
	const buttonsUninitialized = uint32(0xFFFFFFFF)
	prevButtons.Store(buttonsUninitialized)

	onReport := func(rep hidpkg.InputReport) {
		if *dump {
			logger.Printf("report axis=0x%02X buttons=0x%04X", rep.ScratchAxis, rep.Buttons)
			return
		}
		// Scratch axis -> Pulse state machine.
		dir := detector.Update(rep.ScratchAxis)
		if dir != rotation.DirNone {
			pulser.Rotate(dir)
		}
		// Buttons: edge-triggered key events.
		curr := uint32(rep.Buttons)
		prev := prevButtons.Swap(curr)
		if prev == buttonsUninitialized {
			prev = curr // suppress fake transitions on the first report
		}
		changed := prev ^ curr
		for i := 0; i < 16; i++ {
			mask := uint32(1) << i
			if changed&mask == 0 {
				continue
			}
			vk := cfg.ButtonVKMap[i]
			if vk == 0 {
				continue
			}
			isDown := curr&mask != 0
			if err := sender.Send(uint16(vk), isDown); err != nil {
				logger.Printf("sendinput button: %v", err)
			}
		}
	}

	reader := &hidpkg.Reader{
		VID: cfg.Device.VID,
		PID: cfg.Device.PID,
		Parser: hidpkg.Parser{
			ScratchIdx: cfg.Device.ScratchAxisByteIndex,
			BtnStart:   cfg.Device.ButtonsByteRange[0],
			BtnEnd:     cfg.Device.ButtonsByteRange[1],
			ReportID:   cfg.Device.ReportID,
		},
		OnReport: onReport,
		OnReconnect: func() {
			detector.Reset()
			prevButtons.Store(buttonsUninitialized)
		},
		Logger: logger,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-sigCh
		fmt.Fprintf(os.Stderr, "\nreceived %s, shutting down\n", s)
		cancel()
	}()

	reader.Run(ctx)
	logger.Printf("scratch-bridge stopped")
}
