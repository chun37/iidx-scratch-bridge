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
	"sort"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	upstreamhid "github.com/bearsh/hid"

	"github.com/chun37/iidx-scratch-bridge/internal/config"
	hidpkg "github.com/chun37/iidx-scratch-bridge/internal/hid"
	"github.com/chun37/iidx-scratch-bridge/internal/keyout"
	"github.com/chun37/iidx-scratch-bridge/internal/pulse"
	"github.com/chun37/iidx-scratch-bridge/internal/rotation"
)

func main() {
	configPath := flag.String("config", "config.toml", "path to config TOML")
	dump := flag.Bool("dump", false, "log raw Input Reports as hex (probe mode; ignores buttons_byte_range / scratch_axis_byte_index)")
	list := flag.Bool("list", false, "list all attached HID devices with VID/PID and exit (no config required)")
	flag.Parse()

	logger := log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)

	if *list {
		if err := runList(); err != nil {
			logger.Fatalf("list: %v", err)
		}
		return
	}

	if !filepath.IsAbs(*configPath) {
		if cwd, err := os.Getwd(); err == nil {
			*configPath = filepath.Join(cwd, *configPath)
		}
	}

	logger.Printf("scratch-bridge starting; config=%s dump=%v", *configPath, *dump)

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatalf("config: %v", err)
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
		Logger: logger,
	}

	if *dump {
		reader.OnRawReport = func(buf []byte) {
			logger.Printf("report len=%d %s", len(buf), hexString(buf))
		}
	} else {
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
		// Sentinel so the very first report cannot look like every
		// button just changed. We adopt the first report itself as the
		// baseline.
		const buttonsUninitialized = uint32(0xFFFFFFFF)
		prevButtons.Store(buttonsUninitialized)

		reader.OnReport = func(rep hidpkg.InputReport) {
			dir := detector.Update(rep.ScratchAxis)
			if dir != rotation.DirNone {
				pulser.Rotate(dir)
			}
			curr := uint32(rep.Buttons)
			prev := prevButtons.Swap(curr)
			if prev == buttonsUninitialized {
				prev = curr
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
		reader.OnReconnect = func() {
			detector.Reset()
			prevButtons.Store(buttonsUninitialized)
		}
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

// runList prints every HID device the OS exposes, grouped by VID/PID,
// with enough fields to identify the right controller and copy the
// hex IDs into config.toml.
func runList() error {
	if !upstreamhid.Supported() {
		return fmt.Errorf("HID is not supported on this build (CGo disabled?)")
	}
	infos := upstreamhid.Enumerate(0, 0)
	if len(infos) == 0 {
		fmt.Println("(no HID devices found)")
		return nil
	}
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].VendorID != infos[j].VendorID {
			return infos[i].VendorID < infos[j].VendorID
		}
		if infos[i].ProductID != infos[j].ProductID {
			return infos[i].ProductID < infos[j].ProductID
		}
		return infos[i].Interface < infos[j].Interface
	})
	for _, info := range infos {
		fmt.Printf("VID=0x%04X PID=0x%04X iface=%d  %s / %s%s\n",
			info.VendorID, info.ProductID, info.Interface,
			nonEmpty(info.Manufacturer, "?"),
			nonEmpty(info.Product, "?"),
			serialSuffix(info.Serial),
		)
	}
	return nil
}

func nonEmpty(s, fallback string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	return s
}

func serialSuffix(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return "  serial=" + s
}

func hexString(b []byte) string {
	var sb strings.Builder
	sb.Grow(len(b) * 3)
	for i, x := range b {
		if i > 0 {
			sb.WriteByte(' ')
		}
		const hex = "0123456789ABCDEF"
		sb.WriteByte(hex[x>>4])
		sb.WriteByte(hex[x&0x0F])
	}
	return sb.String()
}
