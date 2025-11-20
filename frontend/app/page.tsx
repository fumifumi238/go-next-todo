// app/page.tsx

// 環境変数 NEXT_PUBLIC_API_URL を利用
// サーバーコンポーネントで実行される場合は 'http://backend:8080' を使うのが確実
const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function getData() {
  const res = await fetch(`${API_URL}/api/dbcheck`, {
    // サーバーコンポーネントでのデータキャッシュ設定 (必要に応じて)
    cache: "no-store",
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch data: ${res.statusText}`);
  }

  // GinのハンドラーはJSONを返しているため、.json()でパース
  return res.json();
}

export default async function Page() {
  const data = await getData();

  return (
    <div>
      <h1>CORS Test Page (Gin Backend)</h1>
      <p>Backend Response:</p>
      {/* 取得したJSONデータを表示 */}
      <pre>{JSON.stringify(data, null, 2)}</pre>
      <p>変えた</p>
    </div>
  );
}
