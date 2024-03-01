package main

import (
	"github.com/leaanthony/clir"
	"github.com/pterm/pterm"
	"os"
)

func main() {
	app := clir.NewCli("ChromiumGui", "Go/HTML Appkit", "1.0")

	app.NewSubCommandFunction("build", "Builds the application", buildApplication)
	app.NewSubCommandFunction("dev", "Runs the application in development mode", devApplication)
	app.NewSubCommandFunction("generate", "Generate Bind", devApplication)

	err := app.Run()
	if err != nil {
		pterm.Println()
		pterm.Error.Println(err.Error())
		os.Exit(1)
	}
}
