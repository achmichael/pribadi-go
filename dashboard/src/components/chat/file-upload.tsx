"use client";

import { useRef } from "react";
import { Paperclip, X, FileText, Image } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

export interface AttachedFile {
  file: File;
  previewUrl: string | null;
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function isImageType(file: File): boolean {
  return file.type.startsWith("image/");
}

interface FileUploadProps {
  attachedFile: AttachedFile | null;
  onFileSelect: (file: AttachedFile | null) => void;
  disabled?: boolean;
}

export function FileUpload({ attachedFile, onFileSelect, disabled }: FileUploadProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const previewUrl = isImageType(file) ? URL.createObjectURL(file) : null;
    onFileSelect({ file, previewUrl });

    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  };

  const handleRemove = () => {
    if (attachedFile?.previewUrl) {
      URL.revokeObjectURL(attachedFile.previewUrl);
    }
    onFileSelect(null);
  };

  return (
    <div className="flex items-center gap-2">
      <input
        type="file"
        className="hidden"
        ref={fileInputRef}
        onChange={handleFileChange}
        accept=".txt,.md,.pdf,.docx,.png,.jpg,.jpeg,.webp"
      />

      <Tooltip>
        <TooltipTrigger asChild>
          <button
            className="inline-flex cursor-pointer rounded-md h-7 w-7 items-center justify-center text-zinc-500 shrink-0 hover:bg-zinc-800 hover:text-zinc-300 transition-colors"
            onClick={(e) => {
              e.preventDefault();
              if (!disabled) fileInputRef.current?.click();
            }}
            disabled={disabled}
            type="button"
          >
            <Paperclip className="h-3.5 w-3.5" />
          </button>
        </TooltipTrigger>
        <TooltipContent side="top" className="text-xs font-mono">Attach File</TooltipContent>
      </Tooltip>

      {attachedFile && (
        <div className="flex items-center gap-2 px-2 py-1 rounded-md border border-indigo-500/20 bg-indigo-500/5 max-w-[200px]">
          {attachedFile.previewUrl ? (
            <img
              src={attachedFile.previewUrl}
              alt="preview"
              className="h-6 w-6 rounded object-cover shrink-0 border border-indigo-500/20"
            />
          ) : (
            <div className="h-6 w-6 rounded bg-indigo-500/10 flex items-center justify-center shrink-0 border border-indigo-500/20">
              <FileText className="h-3 w-3 text-indigo-400" />
            </div>
          )}
          <div className="flex flex-col min-w-0 pr-1">
            <span className="text-[10px] text-indigo-200 truncate font-medium">
              {attachedFile.file.name}
            </span>
            <span className="text-[9px] text-indigo-400/60 font-mono">
              {formatFileSize(attachedFile.file.size)}
            </span>
          </div>
          <button
            type="button"
            onClick={handleRemove}
            className="shrink-0 p-0.5 rounded-sm hover:bg-indigo-500/20 text-indigo-400 transition-colors ml-auto"
          >
            <X className="h-3 w-3" />
          </button>
        </div>
      )}
    </div>
  );
}
