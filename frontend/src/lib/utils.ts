import { clsx, type ClassValue } from "clsx"
import { toast } from "sonner"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function showToast(message: string, type: "success" | "error" = "success") {
  if (type === "success") {
    toast.success(message, {
      position: "top-center",
    })
    return
  }
  if (type === "error") {
    toast.error(message, {

      position: "top-center",
    })
    return
  }
}
