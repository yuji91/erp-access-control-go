#!/bin/bash

# 🧪 API動作確認スクリプト
# 複数ロール対応システムのAPI動作確認

set -e

# 設定
API_BASE_URL="http://localhost:8080"
API_VERSION="v1"

# 色付きログ関数
log_info() {
    echo -e "\033[34m[INFO]\033[0m $1"
}

log_success() {
    echo -e "\033[32m[SUCCESS]\033[0m $1"
}

log_error() {
    echo -e "\033[31m[ERROR]\033[0m $1"
}

log_warning() {
    echo -e "\033[33m[WARNING]\033[0m $1"
}

# ヘルスチェック
test_health_check() {
    log_info "🔍 ヘルスチェックテスト"
    
    response=$(curl -s -w "\n%{http_code}" "${API_BASE_URL}/health")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "200" ]; then
        log_success "ヘルスチェック成功 (HTTP $http_code)"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        log_error "ヘルスチェック失敗 (HTTP $http_code)"
        echo "$body"
        return 1
    fi
    echo
}

# バージョン情報
test_version() {
    log_info "📋 バージョン情報テスト"
    
    response=$(curl -s -w "\n%{http_code}" "${API_BASE_URL}/version")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "200" ]; then
        log_success "バージョン情報取得成功 (HTTP $http_code)"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        log_error "バージョン情報取得失敗 (HTTP $http_code)"
        echo "$body"
        return 1
    fi
    echo
}

# ルートエンドポイント
test_root() {
    log_info "🏠 ルートエンドポイントテスト"
    
    response=$(curl -s -w "\n%{http_code}" "${API_BASE_URL}/")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "200" ]; then
        log_success "ルートエンドポイント成功 (HTTP $http_code)"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        log_error "ルートエンドポイント失敗 (HTTP $http_code)"
        echo "$body"
        return 1
    fi
    echo
}

# ログインテスト
test_login() {
    log_info "🔐 ログインテスト"
    
    # テスト用ユーザー情報（seeds/01_test_data.sqlに基づく）
    # 利用可能なテストユーザー:
    # - admin@example.com / password123 (システム管理者)
    # - it-manager@example.com / password123 (IT部門長)
    # - hr-manager@example.com / password123 (人事部長)
    # - developer-a@example.com / password123 (開発者A)
    # - developer-b@example.com / password123 (開発者B)
    # - pm-tanaka@example.com / password123 (PM田中)
    # - user-a@example.com / password123 (一般ユーザーA)
    # - user-b@example.com / password123 (一般ユーザーB)
    # - guest@example.com / password123 (ゲストユーザー)
    local email="admin@example.com"
    local password="password123"
    
    response=$(curl -s -w "\n%{http_code}" \
        -X POST "${API_BASE_URL}/api/${API_VERSION}/auth/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$email\",
            \"password\": \"$password\"
        }")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "200" ]; then
        log_success "ログイン成功 (HTTP $http_code)"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
        
        # アクセストークンを保存
        ACCESS_TOKEN=$(echo "$body" | jq -r '.access_token' 2>/dev/null)
        if [ "$ACCESS_TOKEN" != "null" ] && [ -n "$ACCESS_TOKEN" ]; then
            log_success "アクセストークン取得成功"
            export ACCESS_TOKEN
        else
            log_warning "アクセストークンが取得できませんでした"
        fi
    else
        log_error "ログイン失敗 (HTTP $http_code)"
        echo "$body"
        return 1
    fi
    echo
}

# プロフィール取得テスト
test_profile() {
    log_info "👤 プロフィール取得テスト"
    
    if [ -z "$ACCESS_TOKEN" ]; then
        log_warning "アクセストークンがありません。ログインテストを先に実行してください。"
        return 1
    fi
    
    response=$(curl -s -w "\n%{http_code}" \
        -X GET "${API_BASE_URL}/api/${API_VERSION}/auth/profile" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "200" ]; then
        log_success "プロフィール取得成功 (HTTP $http_code)"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        log_error "プロフィール取得失敗 (HTTP $http_code)"
        echo "$body"
        return 1
    fi
    echo
}

# ユーザーロール一覧取得テスト
test_user_roles() {
    log_info "👥 ユーザーロール一覧取得テスト"
    
    if [ -z "$ACCESS_TOKEN" ]; then
        log_warning "アクセストークンがありません。ログインテストを先に実行してください。"
        return 1
    fi
    
    # テスト用ユーザーID（seeds/01_test_data.sqlに基づく）
    local user_id="880e8400-e29b-41d4-a716-446655440001"
    
    response=$(curl -s -w "\n%{http_code}" \
        -X GET "${API_BASE_URL}/api/${API_VERSION}/users/${user_id}/roles?active=true" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "200" ]; then
        log_success "ユーザーロール一覧取得成功 (HTTP $http_code)"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        log_error "ユーザーロール一覧取得失敗 (HTTP $http_code)"
        echo "$body"
        return 1
    fi
    echo
}

# ロール割り当てテスト
test_assign_role() {
    log_info "➕ ロール割り当てテスト"
    
    if [ -z "$ACCESS_TOKEN" ]; then
        log_warning "アクセストークンがありません。ログインテストを先に実行してください。"
        return 1
    fi
    
    # テスト用データ（seeds/01_test_data.sqlに基づく）
    local user_id="880e8400-e29b-41d4-a716-446655440001"
    local role_id="660e8400-e29b-41d4-a716-446655440003"  # 一般ユーザーロール（重複回避）
    
    response=$(curl -s -w "\n%{http_code}" \
        -X POST "${API_BASE_URL}/api/${API_VERSION}/users/roles" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{
            \"user_id\": \"$user_id\",
            \"role_id\": \"$role_id\",
            \"priority\": 2,
            \"reason\": \"テスト用ロール割り当て\"
        }")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "201" ]; then
        log_success "ロール割り当て成功 (HTTP $http_code)"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        log_error "ロール割り当て失敗 (HTTP $http_code)"
        echo "$body"
        return 1
    fi
    echo
}

# エラーハンドリングテスト
test_error_handling() {
    log_info "⚠️ エラーハンドリングテスト"
    
    # 無効なログイン
    log_info "無効な認証情報でのログイン"
    response=$(curl -s -w "\n%{http_code}" \
        -X POST "${API_BASE_URL}/api/${API_VERSION}/auth/login" \
        -H "Content-Type: application/json" \
        -d '{
            "email": "invalid@example.com",
            "password": "wrongpassword"
        }')
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "401" ]; then
        log_success "無効な認証情報の処理成功 (HTTP $http_code)"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        log_warning "無効な認証情報の処理が期待と異なります (HTTP $http_code)"
        echo "$body"
    fi
    echo
    
    # 無効なトークンでのプロフィール取得
    log_info "無効なトークンでのプロフィール取得"
    response=$(curl -s -w "\n%{http_code}" \
        -X GET "${API_BASE_URL}/api/${API_VERSION}/auth/profile" \
        -H "Authorization: Bearer invalid.token.here")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "401" ]; then
        log_success "無効なトークンの処理成功 (HTTP $http_code)"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        log_warning "無効なトークンの処理が期待と異なります (HTTP $http_code)"
        echo "$body"
    fi
    echo
}

# メイン実行関数
main() {
    log_info "🚀 API動作確認開始"
    log_info "API Base URL: $API_BASE_URL"
    log_info "API Version: $API_VERSION"
    echo
    
    # 基本テスト
    test_health_check
    test_version
    test_root
    
    # 認証テスト
    test_login
    
    # 認証が必要なテスト
    if [ -n "$ACCESS_TOKEN" ]; then
        test_profile
        test_user_roles
        test_assign_role
    fi
    
    # エラーハンドリングテスト
    test_error_handling
    
    log_success "✅ API動作確認完了"
}

# スクリプト実行
main "$@" 