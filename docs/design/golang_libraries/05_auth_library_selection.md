# ERP向けアクセス制御API - 認証・ユーザー管理ライブラリ選定

## 5. 認証・ユーザー管理

| ライブラリ | 用途 | 備考 |
|-----------|------|------|
| golang-jwt/jwt | JWTベースの認証 | ロール・部門IDをJWTに含めることで、Contextに注入可能 |
| go-chi/jwtauth | chiベースJWT Middleware | Ginでも類似のものあり。アクセストークンの検証に使用 |

**ではどちらが適切？**

ERPシステムのように JWTベースで「ロール・部門ID」によるアクセス制御を行う構成 においては、Ginを採用している前提では `golang-jwt/jwt` を直接使うのが最適です。

## ✅ 結論：Gin × JWT構成では golang-jwt/jwt を直接使うのが適切

### 🔍 ライブラリ比較表

| ライブラリ | 用途 | Ginとの親和性 | 柔軟性 | コメント |
|-----------|------|-------------|--------|----------|
| golang-jwt/jwt | JWTトークンの生成・検証 | ◎ | ◎ | Ginに限らず使用できる。カスタムクレーム構造で柔軟に設計可。 |
| go-chi/jwtauth | JWT認証のchi用Middleware | △（Gin用ではない） | ◯（chi限定の抽象） | chi専用のラッパー。Ginとは構文や設計思想が異なる。 |

## ✳️ Gin × JWT構成の最適解

Ginでは、`golang-jwt/jwt` と `Context.Set()` / `Context.Get()` を組み合わせて以下のように構成します：

### ① JWTトークンの構造（カスタムクレーム）

```go
type CustomClaims struct {
  UserID     string `json:"user_id"`
  Department string `json:"department"`
  Role       string `json:"role"`
  jwt.StandardClaims
}
```

### ② JWT Middlewareの実装例

```go
func JWTMiddleware(secret []byte) gin.HandlerFunc {
  return func(c *gin.Context) {
    tokenString := extractToken(c.Request) // ヘッダー等から取得
    token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
      return secret, nil
    })

    if err != nil || !token.Valid {
      c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
      return
    }

    claims := token.Claims.(*CustomClaims)
    c.Set("user_id", claims.UserID)
    c.Set("role", claims.Role)
    c.Set("department", claims.Department)

    c.Next()
  }
}
```

### ③ ハンドラでの取得

```go
func GetPermissions(c *gin.Context) {
  userID := c.GetString("user_id")
  role := c.GetString("role")
  dept := c.GetString("department")

  // DBクエリやマトリクス評価に利用可能
}
```

## ✅ なぜ go-chi/jwtauth は不適か？

- `go-chi/jwtauth` は chi用に最適化された設計
- `http.Handler` ベースで `gin.Context` に合わない
- `jwtauth.Verifier()` や `jwtauth.Authenticator()` などはGinで扱うには冗長
- Ginには `gin-jwt` や `jwt-go` ベースの軽量ミドルウェアを自作する構成が主流

## 🔧 Gin用の代替 JWTライブラリ（補足）

| ライブラリ | 概要 | 備考 |
|-----------|------|------|
| appleboy/gin-jwt/v2 | Gin専用のJWTミドルウェア | ログイン〜トークン発行〜認証まで一式あり。トークンにカスタムクレーム追加も容易。 |
| gin-contrib/sessions + jwt | セッションベースと併用可 | セッションとの併用やリフレッシュトークン管理にも対応 |

## ✅ まとめ

| 選択肢 | 採用可否 | 理由 |
|-------|----------|------|
| golang-jwt/jwt | ✅ 採用推奨 | Ginと親和性が高く、ロールや部門などの任意情報を柔軟に含められる |
| go-chi/jwtauth | ❌ 不適 | chi専用設計のため、Ginとは設計思想が異なる |

ご希望あれば、JWT生成コード・トークン発行エンドポイント・リフレッシュトークン管理の例も提示可能です。お気軽にどうぞ。
