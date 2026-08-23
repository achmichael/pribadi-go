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
        <TooltipTrigger>
          <div
            className="inline-flex cursor-pointer rounded-xl h-9 w-9 items-center justify-center text-zinc-400 shrink-0 hover:bg-zinc-800 hover:text-zinc-200 transition-colors"
            onClick={(e) => {
              e.preventDefault();
              if (!disabled) fileInputRef.current?.click();
            }}
            data-disabled={disabled}
          >
            <Paperclip className="h-4 w-4" />
          </div>
        </TooltipTrigger>
        <TooltipContent side="top">Attach file</TooltipContent>
      </Tooltip>

      {attachedFile && (
        <div className="flex items-center gap-2 px-2.5 py-1 rounded-lg border border-zinc-800 bg-zinc-900/50 max-w-[240px]">
          {attachedFile.previewUrl ? (
            <img
              src={attachedFile.previewUrl}
              alt="preview"
              className="h-8 w-8 rounded object-cover shrink-0"
            />
          ) : (
            <div className="h-8 w-8 rounded bg-zinc-800 flex items-center justify-center shrink-0">
              <FileText className="h-4 w-4 text-zinc-400" />
            </div>
          )}
          <div className="flex flex-col min-w-0">
            <span className="text-[11px] text-zinc-300 truncate font-medium">
              {attachedFile.file.name}
            </span>
            <span className="text-[10px] text-zinc-500">
              {formatFileSize(attachedFile.file.size)}
            </span>
          </div>
          <button
            onClick={handleRemove}
            className="shrink-0 p-0.5 rounded hover:bg-zinc-700 transition-colors"
          >
            <X className="h-3 w-3 text-zinc-400" />
          </button>
        </div>
      )}
    </div>
  );
}
