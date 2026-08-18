import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"
import "./keyboard-shortcut-guard"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
