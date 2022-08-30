package main

import (
    "fmt"
    "github.com/common-nighthawk/go-figure"
    "github.com/sirupsen/logrus"
    "github.com/spf13/cobra"
    proxy "mixed-socks"
    "os"
    "os/signal"
    "path"
    "strings"
    "syscall"
)

var (
    host   string
    port   int
    Header = figure.NewFigure("MixedSocks", "doom", true).String()
    cmd    = &cobra.Command{
        Use:               os.Args[0],
        Short:             "Support socks4, socks4a, socks5, socks5h, http proxy all in one",
        DisableAutoGenTag: true,
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Println(Header)
            server := proxy.NewSocksServer(host, port)
            server.ListenAndServe()
        },
    }
)

func init() {
    logrus.SetLevel(logrus.DebugLevel)
    logrus.SetReportCaller(true)
    logrus.SetFormatter(&ConsoleFormatter{})
    registerSignalHandlers()
    cmd.PersistentFlags().StringVarP(&host, "addr", "a", "localhost", "listen addr")
    cmd.PersistentFlags().IntVarP(&port, "port", "p", 1080, "listen port")
}

func main() {
    cobra.CheckErr(cmd.Execute())
}

func registerSignalHandlers() {
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
    go func() {
        <-sigs
        os.Exit(0)
    }()
}

type ConsoleFormatter struct {
    logrus.TextFormatter
}

func (c *ConsoleFormatter) Format(entry *logrus.Entry) ([]byte, error) {
    logStr := fmt.Sprintf("%s %s %s:%d %v\n",
        entry.Time.Format("2006/01/02 15:04:05"),
        strings.ToUpper(entry.Level.String()),
        path.Base(entry.Caller.File),
        entry.Caller.Line,
        entry.Message,
    )
    return []byte(logStr), nil
}
