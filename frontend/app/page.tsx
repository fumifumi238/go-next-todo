"use client"; // 💡 App Routerでフック(useState, useEffect)を使うために必須

import { useState, useEffect } from "react";
import Head from "next/head";

// APIから返されるデータの型定義
interface ApiResponse {
  message: string;
  service: string;
}

// Next.jsの環境変数からAPIのベースURLを取得
// docker-compose.ymlで NEXT_PUBLIC_API_URL=http://golang:8080 に設定されています
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL;

const Home: React.FC = () => {
  const [data, setData] = useState<ApiResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchData() {
      if (!API_BASE_URL) {
        setError("API URL is not defined in environment variables.");
        setLoading(false);
        return;
      }

      try {
        // Go APIのエンドポイントを呼び出す
        const response = await fetch(`${API_BASE_URL}/api/hello`);

        if (!response.ok) {
          throw new Error(`API returned status ${response.status}`);
        }

        const result: ApiResponse = await response.json();
        setData(result);
      } catch (e: unknown) {
        if (e instanceof Error) {
          setError(
            `Failed to fetch data: ${e.message}. Check if Go API is running.`
          );
        }

        console.error("Fetch Error:", e);
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, []);

  return (
    <div style={{ padding: "20px", fontFamily: "Arial, sans-serif" }}>
      <Head>
        <title>Next.js & Go Connection Test</title>
      </Head>

      <h1>Next.js 🤝 Golang 接続テスト</h1>

      {loading && <p>Loading API response...</p>}

      {error && (
        <div style={{ color: "red", border: "1px solid red", padding: "10px" }}>
          <h2>❌ Connection Error</h2>
          <p>{error}</p>
          <p>Target URL: {API_BASE_URL}/api/hello</p>
          <p>💡 Make sure the Go API service is running correctly in Docker.</p>
        </div>
      )}

      {data && (
        <div
          style={{
            border: "1px solid green",
            padding: "15px",
            backgroundColor: "#e8ffe8",
          }}>
          <h2>✅ Successful Connection!</h2>
          <p>
            <strong>Response Source:</strong> {data.service}
          </p>
          <p>
            <strong>Message:</strong> {data.message}
          </p>
          <p>
            This confirms that the **Next.js (Vercel)** service successfully
            communicated with the **Go API (Fargate)** service within the Docker
            Compose network.
          </p>
        </div>
      )}
    </div>
  );
};

export default Home;
