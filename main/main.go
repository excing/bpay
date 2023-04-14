package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"com.blendiv.pay/ent"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v4/stdlib"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	PGSQL string `json:"pgsql"`
}

var config *Config
var ctx = context.Background()

func main() {
	var configFilepath string
	flag.StringVar(&configFilepath, "config", "Server config file path", "")
	flag.Parse()
	flag.Usage()
	if "" != configFilepath {
		bs, err := os.ReadFile(configFilepath)
		if err == nil {
			if err = json.Unmarshal(bs, &config); err != nil {
				// log.Panicln("json unmarshal config file failed: %v", err)
				panic(err)
			}
		}
	}

	db, err := sql.Open("pgx", "postgresql://"+config.PGSQL)
	if err != nil {
		log.Fatalf("failed opening connection to postgres: %v", err)
	}
	defer db.Close()

	drv := entsql.OpenDB(dialect.Postgres, db)

	opts := []ent.Option{
		ent.Driver(drv),
		ent.Debug(),
	}

	client := ent.NewClient(opts...)

	// Run the auto migration tool.
	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	router := gin.Default()

	server := &http.Server{
		Addr:         ":10090",
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	var g errgroup.Group
	g.Go(func() error {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
		return err
	})

	router.Use(gzip.Gzip(gzip.DefaultCompression))

	router.GET("create", func(c *gin.Context) {
		c.String(200, "create order id")
	})

	if err = g.Wait(); err != nil {
		panic(err)
	}

}

func credits(c *gin.Context) {

}
