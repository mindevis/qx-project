import { vi } from 'vitest';

function create2dContext(canvas: HTMLCanvasElement): CanvasRenderingContext2D {
  return {
    canvas,
    fillRect: vi.fn(),
    clearRect: vi.fn(),
    getImageData: vi.fn(),
    putImageData: vi.fn(),
    createImageData: vi.fn(() => ({ data: new Uint8ClampedArray(4), width: 1, height: 1 })),
    setTransform: vi.fn(),
    resetTransform: vi.fn(),
    drawImage: vi.fn(),
    save: vi.fn(),
    restore: vi.fn(),
    scale: vi.fn(),
    rotate: vi.fn(),
    translate: vi.fn(),
    transform: vi.fn(),
    measureText: vi.fn(() => ({ width: 0 })),
    fillText: vi.fn(),
    strokeText: vi.fn(),
    beginPath: vi.fn(),
    closePath: vi.fn(),
    moveTo: vi.fn(),
    lineTo: vi.fn(),
    stroke: vi.fn(),
    fill: vi.fn(),
    clip: vi.fn(),
    createLinearGradient: vi.fn(),
    createRadialGradient: vi.fn(),
    createPattern: vi.fn(),
    arc: vi.fn(),
    rect: vi.fn(),
    quadraticCurveTo: vi.fn(),
    bezierCurveTo: vi.fn(),
    globalAlpha: 1,
    globalCompositeOperation: 'source-over',
  } as unknown as CanvasRenderingContext2D;
}

function createWebGLContext(canvas: HTMLCanvasElement): WebGLRenderingContext {
  const noop = vi.fn();
  return {
    canvas,
    getExtension: vi.fn(() => null),
    getParameter: vi.fn(() => 0),
    createShader: vi.fn(() => ({})),
    shaderSource: noop,
    compileShader: noop,
    createProgram: vi.fn(() => ({})),
    attachShader: noop,
    linkProgram: noop,
    useProgram: noop,
    createBuffer: vi.fn(() => ({})),
    bindBuffer: noop,
    bufferData: noop,
    viewport: noop,
    clear: noop,
    clearColor: noop,
    enable: noop,
    disable: noop,
    blendFunc: noop,
    pixelStorei: noop,
    drawArrays: noop,
    drawElements: noop,
    getShaderParameter: vi.fn(() => true),
    getProgramParameter: vi.fn(() => true),
    getAttribLocation: vi.fn(() => 0),
    getUniformLocation: vi.fn(() => ({})),
    uniform1i: noop,
    uniform1f: noop,
    uniformMatrix4fv: noop,
    vertexAttribPointer: noop,
    enableVertexAttribArray: noop,
    activeTexture: noop,
    bindTexture: noop,
    texImage2D: noop,
    texParameteri: noop,
    generateMipmap: noop,
    createTexture: vi.fn(() => ({})),
    deleteTexture: noop,
    deleteBuffer: noop,
    deleteProgram: noop,
    deleteShader: noop,
  } as unknown as WebGLRenderingContext;
}

/** jsdom logs without the optional `canvas` npm package; three.js/skinview3d need WebGL. */
export function installCanvasMocks() {
  HTMLCanvasElement.prototype.getContext = vi.fn(function getContext(
    this: HTMLCanvasElement,
    contextId: string,
  ) {
    if (contextId === '2d') {
      return create2dContext(this);
    }
    if (contextId === 'webgl' || contextId === 'webgl2' || contextId === 'experimental-webgl') {
      return createWebGLContext(this);
    }
    return null;
  }) as typeof HTMLCanvasElement.prototype.getContext;

  if (!HTMLCanvasElement.prototype.toDataURL) {
    HTMLCanvasElement.prototype.toDataURL = vi.fn(() => 'data:image/png;base64,');
  }
}
