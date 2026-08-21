"use client";

import { useRef, useState } from "react";
import { Paperclip, Loader2, CheckCircle2, XCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { fetchApi } from "@/lib/api";

export function FileUpload() {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [status, setStatus] = useState<"idle" | "uploading" | "success" | "error">("idle");
  
  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setStatus("uploading");
    
    const formData = new FormData();
    formData.append("file", file);

    try {
      const res = await fetch(`${process.env.NODE_ENV === 'development' ? 'http://localhost:8090/api/v1' : '/api/v1'}/chat/upload`, {
        method: "POST",
        headers: {
          ...(typeof window !== 'undefined' && localStorage.getItem('token') ? { 'Authorization': `Bearer ${localStorage.getItem('token')}` } : {})
        },
        body: formData
      });
      
      if (!res.ok) throw new Error("Upload failed");
      setStatus("success");
      setTimeout(() => setStatus("idle"), 3000);
    } catch (err) {
      console.error(err);
      setStatus("error");
      setTimeout(() => setStatus("idle"), 3000);
    }
  };

  return (
    <>
      <input 
        type="file" 
        className="hidden" 
        ref={fileInputRef} 
        onChange={handleUpload}
        accept=".txt,.md,.pdf,.docx"
      />
      
      <Tooltip>
        <TooltipTrigger>
          <div 
            className="inline-flex cursor-pointer rounded-xl h-9 w-9 items-center justify-center text-zinc-400 shrink-0 hover:bg-zinc-800 hover:text-zinc-200 transition-colors"
            onClick={(e) => {
              e.preventDefault();
              if (status !== "uploading") {
                fileInputRef.current?.click();
              }
            }}
            data-disabled={status === "uploading"}
          >
            {status === "idle" && <Paperclip className="h-4 w-4" />}
            {status === "uploading" && <Loader2 className="h-4 w-4 animate-spin text-primary" />}
            {status === "success" && <CheckCircle2 className="h-4 w-4 text-emerald-500" />}
            {status === "error" && <XCircle className="h-4 w-4 text-destructive" />}
          </div>
        </TooltipTrigger>
        <TooltipContent side="top">
          {status === "idle" ? "Upload document for RAG" : 
           status === "uploading" ? "Uploading and processing..." :
           status === "success" ? "Added to context" : "Failed to upload"}
        </TooltipContent>
      </Tooltip>
    </>
  );
}
