# ベースイメージを指定
FROM node:20-alpine

# 作業ディレクトリを設定
WORKDIR /app

# キャッシュ利用で効率化するために別でコピー
COPY ./frontend/package*.json ./

# 依存関係をインストール
RUN npm install

# ソースコードをコピー
COPY ./frontend/ ./app

# アプリケーションをビルド
RUN npm run build

# ポート番号を指定
EXPOSE 3000

# アプリケーションを起動
CMD ["npm", "start"]
