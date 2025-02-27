package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/OpenDiablo2/HellSpawner/hscommon"
	"github.com/ianling/giu"

	lib "github.com/gravestench/dcc/pkg"
	"github.com/gravestench/dcc/pkg/giuwidget"
)

const (
	title = "dcc viewer"
	defaultWidth = 256
	defaultHeight = 256
	windowFlags = giu.MasterWindowFlagsFloating & giu.MasterWindowFlagsNotResizable
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	srcPath := os.Args[1]

	fileContents, err := ioutil.ReadFile(srcPath)
	if err != nil {
		const fmtErr = "could not read file, %v"
		fmt.Print(fmt.Errorf(fmtErr, err))

		return
	}

	dcc, err := lib.FromBytes(fileContents)
	if err != nil {
		fmt.Print(err)
		return
	}

	firstFrame := dcc.Direction(0).Frame(0)
	window := giu.NewMasterWindow(title, firstFrame.Width, firstFrame.Height, windowFlags, nil)

	tl := hscommon.NewTextureLoader()

	widget := giuwidget.Create(tl, nil, "dccviewer", dcc)

	window.Run(func() {
		widget.Build()
	})
}

func printUsage() {
	fmt.Printf("Usage:\r\n\t%s path/to/file.lib", os.Args[0])
}

func render() {

}
