declare module "upng-js" {
  const UPNG: {
    /** Encodes RGBA frames as PNG. cnum 0 keeps every colour (lossless); a positive number quantises to that many colours. */
    encode(frames: ArrayBuffer[], width: number, height: number, cnum: number): ArrayBuffer;
  };
  export default UPNG;
}
