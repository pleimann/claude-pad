// Shared types for camel-pad

export interface Config {
  device: {
    port?: string;
    vendorId?: number;
    productId?: number;
  };
  server: {
    port: number;
    host: string;
  };
  gestures: {
    longPressMs: number;
    doublePressMs: number;
  };
  keys: Record<string, KeyMapping>;
  defaults: {
    timeoutMs: number;
  };
  handedness: 'left' | 'right';
}

export interface KeyMapping {
  press?: ActionMapping;
  doublePress?: ActionMapping;
  longPress?: ActionMapping;
}

export interface ActionMapping {
  action: string;
  label: string;
}

// WebSocket message types
export interface NotificationMessage {
  type: 'notification' | 'test' | 'message';
  id: string;
  text: string;
  category?: string;
}

export interface ResponseMessage {
  type: 'response';
  id: string;
  action: string;
  label: string;
}

export interface ErrorMessage {
  type: 'error';
  id: string;
  error: string;
}

export type OutgoingMessage = ResponseMessage | ErrorMessage;

// Serial protocol constants (matches firmware config.h)
export const FRAME_START_BYTE = 0xAA;
export const MAX_MSG_LEN = 512;
export const SERIAL_BAUD = 115200;

export const MSG_DISPLAY_TEXT = 0x01;
export const MSG_BUTTON = 0x02;
export const MSG_SET_LEDS = 0x03;
export const MSG_STATUS = 0x04;
export const MSG_CLEAR = 0x05;
export const MSG_SET_LABELS = 0x06;
export const MSG_HEARTBEAT = 0x07;
export const MSG_PING      = 0x08; // Host→Device: keepalive (no payload)

// Gesture types
export type GestureType = 'press' | 'doublePress' | 'longPress';

export interface GestureEvent {
  buttonId: string;
  gesture: GestureType;
}

// Pending notification in the queue
export interface PendingNotification {
  id: string;
  text: string;
  category?: string;
  timeoutMs: number;
  timeoutHandle: ReturnType<typeof setTimeout>;
  resolve: (response: ResponseMessage) => void;
  reject: (error: Error) => void;
}
