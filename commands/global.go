package commands

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"time"

	"github.com/mylxsw/heimdall/query"
	"github.com/urfave/cli/v2"
)

func readAll(r io.Reader, endSign byte) string {
	reader := bufio.NewReader(r)

	var result []byte
	for {
		input, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			}

			panic(err)
		}

		if input == '\n' && len(result) > 0 && result[len(result)-1] == endSign {
			break
		}

		result = append(result, input)
	}

	return string(result)
}

func BuildGlobalFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "host", Aliases: []string{"H"}, Value: "127.0.0.1", Usage: "MySQL host"},
		&cli.IntFlag{Name: "port", Aliases: []string{"P"}, Value: 3306, Usage: "MySQL port"},
		&cli.StringFlag{Name: "user", Aliases: []string{"u"}, Value: "root", Usage: "MySQL user"},
		&cli.StringFlag{Name: "password", Aliases: []string{"p"}, Value: "", Usage: "MySQL password"},
		&cli.StringFlag{Name: "database", Aliases: []string{"d"}, Value: "", Usage: "MySQL database"},
		&cli.BoolFlag{Name: "debug", Aliases: []string{"D"}, Value: false, Usage: "debug mode"},
		&cli.DurationFlag{Name: "connect-timeout", Value: 3 * time.Second, Usage: "database connect timeout"},
		&cli.BoolFlag{Name: "beta", Usage: "enable beta feature, may be unstable, use at your own risk"},
	}
}

type GlobalOption struct {
	Host           string
	Port           int
	User           string
	Password       string
	Database       string
	Debug          bool
	Beta           bool
	ConnectTimeout time.Duration
}

func (globalOption GlobalOption) DSN() string {
	return query.BuildConnStr(
		globalOption.Database,
		globalOption.User,
		globalOption.Password,
		globalOption.Host,
		globalOption.Port,
	)
}

func resolveGlobalOption(c *cli.Context) GlobalOption {
	return GlobalOption{
		Host:           c.String("host"),
		Port:           c.Int("port"),
		User:           c.String("user"),
		Password:       c.String("password"),
		Database:       c.String("database"),
		Debug:          c.Bool("debug"),
		ConnectTimeout: c.Duration("connect-timeout"),
		Beta:           c.Bool("beta"),
	}
}

func fileHash(filepath string) (string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
