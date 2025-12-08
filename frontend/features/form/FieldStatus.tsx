"use client";

import React from "react";
import { FaCheckCircle, FaTimesCircle } from "react-icons/fa";

type Props = {
  value: string | undefined;
  error?: string;
};

const FieldStatus: React.FC<Props> = ({ value, error }) => {
  // エラーあり → 赤
  if (error) {
    return (
      <div className="min-h-6 mt-1 flex items-center gap-1">
        <FaTimesCircle className="text-red-500" />
        <p className="text-sm text-red-600">{error}</p>
      </div>
    );
  }

  // 未入力なら空の高さ確保だけ
  if (!value) return <div className="min-h-6 mt-1" />;

  // 問題なし → 緑
  return (
    <div className="min-h-6 mt-1 flex items-center gap-1">
      <FaCheckCircle className="text-green-500" />
      <p className="text-sm text-green-600">OK</p>
    </div>
  );
};

export default FieldStatus;
