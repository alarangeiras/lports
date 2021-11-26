package main

import (
	"fmt"
	"log"
	"lports/lsof"

	"github.com/jedib0t/go-pretty/v6/table"
)

var (
	tableHandler   = table.Table{}
	tableHeader = table.Row{"PID #", "Command", "User ID", "PortNumber"}
)

func main() {
	processes, err := lsof.Run()
	if err != nil {
		log.Fatalln(err)
	}
	tableHandler.AppendHeader(tableHeader)
	for _, process := range processes {
		tableHandler.AppendRow(table.Row{
			process.PID,
			process.Command,
			process.UserID,
			process.PortNumber,
		})
	}
	fmt.Println(tableHandler.Render())
}
