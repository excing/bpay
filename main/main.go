package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"com.blendiv.pay/ent"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v4/stdlib"

	openai "github.com/sashabaranov/go-openai"

	"github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth/v7/limiter"
)

// Config 配置
type Config struct {
	Port        int    `json:"port"`
	PGSQL       string `json:"pgsql"`
	OpenAIAPI   string `json:"openaiApi"`
	OpenAIToken string `json:"openaiToken"`
}

var config *Config
var ctx = context.Background()

var openaiClient *openai.Client

func main() {
	var configFilepath string
	flag.StringVar(&configFilepath, "config", "Server config file path", "")
	flag.Parse()
	flag.Usage()
	bs, err := os.ReadFile(configFilepath)
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal(bs, &config); err != nil {
		panic(err)
	}
	if config.Port == 0 {
		panic("Port can't equal 0")
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
	defer client.Close()

	openaiConfig := openai.DefaultConfig(config.OpenAIToken)
	openaiConfig.BaseURL = fmt.Sprintf("%v/v1", config.OpenAIAPI)

	openaiClient = openai.NewClientWithConfig(openaiConfig)

	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 1秒最多允许请求 1次，1年后重置限制器
	limiter := tollbooth.NewLimiter(1, &limiter.ExpirableOptions{DefaultExpirationTTL: 8760 * time.Hour})

	limiter.SetIPLookups([]string{"RemoteAddr", "X-Forwarded-For", "X-Real-IP"})

	router.POST("/v1/chat/completions", chat)
	router.GET("buy", LimitHandler(limiter), buy)
	router.GET("credits", credits)

	addr := fmt.Sprintf(":%d", config.Port)

	router.Run(addr)
}

func LimitHandler(lmt *limiter.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		httpError := tollbooth.LimitByRequest(lmt, c.Writer, c.Request)
		if httpError != nil {
			c.Data(httpError.StatusCode, lmt.GetMessageContentType(), []byte(httpError.Message))
			c.Abort()
		} else {
			c.Next()
		}
	}
}

func chat(c *gin.Context) {
	var req openai.ChatCompletionRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("/chat/completions error: %v", err))
		return
	}

	if req.Stream {
		chatCompletionStream(c, req)
	} else {
		chatCompletion(c, req)
	}

}

func chatCompletion(c *gin.Context, req openai.ChatCompletionRequest) {
	response, err := openaiClient.CreateChatCompletion(ctx, req)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("/chat/completions error: %v", err))
		return
	}
	c.JSON(200, &response)
}

func chatCompletionStream(c *gin.Context, req openai.ChatCompletionRequest) {
	stream, err := openaiClient.CreateChatCompletionStream(ctx, req)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("/chat/completions error: %v", err))
		return
	}
	defer stream.Close()

	chanStream := make(chan string, 10)
	go func() {
		defer close(chanStream)
		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				chanStream <- "[DONE]"
				return
			}

			if err != nil {
				fmt.Printf("\nStream error: %v\n", err)
				return
			}

			respJSONBytes, _ := json.Marshal(response)

			chanStream <- string(respJSONBytes)
		}
	}()
	c.Stream(func(w io.Writer) bool {
		if msg, ok := <-chanStream; ok {
			c.SSEvent("message", msg)
			return true
		}
		return false
	})
}

func buy(c *gin.Context) {
	fee := QueryDefaultIntByGinContext(c, "fee", 0)
	cip := c.ClientIP()
	// cfConnectingIP := c.GetHeader("CF-Connecting-IP")
	// xForwardedFor := c.GetHeader("X-Forwarded-For")

	c.JSON(200, gin.H{
		"fee": fee,
		"cip": cip,
	})
}

func createUser(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token != "" || 58 != len(token) {
		c.String(http.StatusBadRequest, "Invailde token")
		return
	}
}

func credits(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" || 58 != len(token) {
		c.String(http.StatusBadRequest, "Invailde token")
		return
	}
}

// QueryDefaultIntByGinContext returns 指定 key 的 int 值
func QueryDefaultIntByGinContext(c *gin.Context, key string, def int) int {
	v, ok := c.GetQuery(key)
	if !ok {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

// QueryIntByGinContext returns 指定 key 的 int 值
func QueryIntByGinContext(c *gin.Context, key string) (int, error) {
	v, ok := c.GetQuery(key)
	if !ok {
		return 0, fmt.Errorf("Missing argument(%v)", key)
	}
	i, err := strconv.Atoi(v)
	return i, err
}
