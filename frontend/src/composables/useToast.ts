import { readonly, ref } from 'vue';

export type ToastType = 'success' | 'error' | 'info' | 'warning';

export interface Toast {
  id: string;
  type: ToastType;
  message: string;
  duration?: number;
}

const DEFAULT_DURATION = 3000;

const toasts = ref<Toast[]>([]);
const timers = new Map<string, ReturnType<typeof setTimeout>>();

let counter = 0;
function nextId() {
  counter += 1;
  return `t${Date.now().toString(36)}-${counter}`;
}

function dismiss(id: string) {
  const timer = timers.get(id);
  if (timer) {
    clearTimeout(timer);
    timers.delete(id);
  }
  toasts.value = toasts.value.filter((t) => t.id !== id);
}

function push(toast: Omit<Toast, 'id'> & { id?: string }) {
  const id = toast.id ?? nextId();
  const duration = toast.duration ?? DEFAULT_DURATION;
  toasts.value = [...toasts.value, { ...toast, id, duration }];
  if (duration > 0) {
    const timer = setTimeout(() => dismiss(id), duration);
    timers.set(id, timer);
  }
  return id;
}

export function useToast() {
  return {
    toasts: readonly(toasts),
    push,
    dismiss,
  };
}
