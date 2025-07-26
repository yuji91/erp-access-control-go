# =============================================================================
# ERP Access Control API - 本番用 Dockerfile
# =============================================================================
# マルチステージビルド：ビルド環境 + 本番実行環境

# ビルドステージ
FROM golang:1.24-alpine AS builder

# メタデータ
LABEL stage=builder

# 必要な開発ツールのインストール
RUN apk add --no-cache \
    git \
    ca-certificates \
    make \
    && update-ca-certificates

# 作業ディレクトリ設定
WORKDIR /app

# Go modulesの設定
ENV GO111MODULE=on
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org

# 依存関係ファイルをコピー
COPY go.mod go.sum ./

# 依存関係ダウンロード
RUN go mod download && go mod verify

# ソースコードをコピー
COPY . .

# アプリケーションビルド
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o bin/erp-access-control-api \
    cmd/server/main.go

# 実行ファイルの権限設定
RUN chmod +x bin/erp-access-control-api

# =============================================================================
# 本番実行ステージ
FROM alpine:3.19 AS production

# メタデータ
LABEL maintainer="ERP Access Control API Team"
LABEL description="Production image for ERP Access Control API"
LABEL version="1.0.0"

# セキュリティ：非rootユーザー作成
RUN addgroup -g 1000 -S erp && \
    adduser -u 1000 -S erp -G erp

# 必要な最小限のパッケージのインストール
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    && update-ca-certificates

# タイムゾーン設定
ENV TZ=Asia/Tokyo
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 作業ディレクトリ作成
WORKDIR /app

# 必要なディレクトリ作成
RUN mkdir -p logs migrations && \
    chown -R erp:erp /app

# ビルドステージから実行ファイルをコピー
COPY --from=builder /app/bin/erp-access-control-api /app/
COPY --from=builder /app/migrations /app/migrations/

# 設定ファイルがある場合はコピー
COPY --from=builder /app/api /app/api/

# 権限設定
RUN chown -R erp:erp /app

# 非rootユーザーに切り替え
USER erp

# ポート公開
EXPOSE 8080

# 環境変数設定
ENV GIN_MODE=release
ENV APP_ENV=production

# ヘルスチェック
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# アプリケーション起動
CMD ["./erp-access-control-api"] 