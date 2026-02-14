import { EventEmitter } from 'events';
import { FRAME_START_BYTE, MAX_MSG_LEN } from '../types.js';

export interface ParsedFrame {
  msgType: number;
  payload: Buffer;
}

function xorChecksum(data: Buffer, offset: number, length: number): number {
  let cs = 0;
  for (let i = offset; i < offset + length; i++) {
    cs ^= data[i];
  }
  return cs;
}

/**
 * Build a binary frame matching the firmware protocol:
 *   [0xAA] [LEN_HI] [LEN_LO] [MSG_TYPE] [PAYLOAD...] [CHECKSUM]
 *
 * LEN = 1 (msgType) + payload.length
 * CHECKSUM = XOR of MSG_TYPE + PAYLOAD bytes
 */
export function buildFrame(msgType: number, payload?: Buffer): Buffer {
  const payloadLen = payload ? payload.length : 0;
  const bodyLen = 1 + payloadLen; // msgType + payload
  const frame = Buffer.alloc(5 + payloadLen);

  frame[0] = FRAME_START_BYTE;
  frame[1] = (bodyLen >> 8) & 0xFF;
  frame[2] = bodyLen & 0xFF;
  frame[3] = msgType;
  if (payload && payloadLen > 0) {
    payload.copy(frame, 4);
  }
  frame[4 + payloadLen] = xorChecksum(frame, 3, bodyLen);

  return frame;
}

const enum ParserState {
  WAIT_START,
  READ_LEN_HI,
  READ_LEN_LO,
  READ_BODY,
  READ_CHECKSUM,
}

/**
 * Streaming frame parser — feed it chunks of serial data,
 * it emits 'frame' events with { msgType, payload }.
 */
export class FrameParser extends EventEmitter {
  private state: ParserState = ParserState.WAIT_START;
  private bodyLen = 0;
  private bodyBuf = Buffer.alloc(MAX_MSG_LEN);
  private bodyPos = 0;

  /** Feed a chunk of incoming serial data. */
  parse(data: Buffer): void {
    for (let i = 0; i < data.length; i++) {
      const byte = data[i];

      switch (this.state) {
        case ParserState.WAIT_START:
          if (byte === FRAME_START_BYTE) {
            this.state = ParserState.READ_LEN_HI;
          }
          break;

        case ParserState.READ_LEN_HI:
          this.bodyLen = byte << 8;
          this.state = ParserState.READ_LEN_LO;
          break;

        case ParserState.READ_LEN_LO:
          this.bodyLen |= byte;
          if (this.bodyLen === 0 || this.bodyLen > MAX_MSG_LEN) {
            this.state = ParserState.WAIT_START;
          } else {
            this.bodyPos = 0;
            this.state = ParserState.READ_BODY;
          }
          break;

        case ParserState.READ_BODY:
          this.bodyBuf[this.bodyPos++] = byte;
          if (this.bodyPos >= this.bodyLen) {
            this.state = ParserState.READ_CHECKSUM;
          }
          break;

        case ParserState.READ_CHECKSUM: {
          const expected = xorChecksum(this.bodyBuf, 0, this.bodyLen);
          if (byte === expected) {
            const msgType = this.bodyBuf[0];
            const payload = Buffer.from(this.bodyBuf.subarray(1, this.bodyLen));
            this.emit('frame', { msgType, payload } as ParsedFrame);
          }
          this.state = ParserState.WAIT_START;
          break;
        }
      }
    }
  }

  reset(): void {
    this.state = ParserState.WAIT_START;
    this.bodyLen = 0;
    this.bodyPos = 0;
  }
}
