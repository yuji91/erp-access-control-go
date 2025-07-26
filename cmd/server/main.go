package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 開発モード設定
	gin.SetMode(gin.DebugMode)

	// Ginルーター初期化
	router := gin.Default()

	// ヘルスチェックエンドポイント
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "erp-access-control-api",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   "0.1.0-dev",
		})
	})

	// バージョン情報エンドポイント
	router.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "ERP Access Control API",
			"version": "0.1.0-dev",
			"status":  "development",
			"message": "API実装準備完了 - 開発フェーズ",
		})
	})

	// ルートエンドポイント
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "🔐 ERP Access Control API",
			"status":  "running",
			"endpoints": []string{
				"GET /health   - ヘルスチェック",
				"GET /version  - バージョン情報",
			},
		})
	})

	// サーバーポート設定
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	// サーバー起動ログ
	log.Printf("🚀 ERP Access Control API サーバー起動中...")
	log.Printf("📡 ポート: %s", port)
	log.Printf("🌐 URL: http://localhost:%s", port)
	log.Printf("🏥 ヘルスチェック: http://localhost:%s/health", port)

	// サーバー起動
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("❌ サーバー起動エラー: %v", err)
	}
}
