export const planetVertexShader = /* glsl */ `
varying vec3 vNormal;
varying vec3 vWorldPosition;
varying vec2 vUv;

void main() {
  vUv = uv;
  vec4 worldPosition = modelMatrix * vec4(position, 1.0);
  vWorldPosition = worldPosition.xyz;
  vNormal = normalize(mat3(modelMatrix) * normal);
  gl_Position = projectionMatrix * viewMatrix * worldPosition;
}
`;

export const planetFragmentShader = /* glsl */ `
uniform float uTime;
uniform vec3 uBaseColor;
uniform vec3 uSurfaceColor;
uniform vec3 uGlowColor;
uniform vec3 uHighlight;
uniform vec3 uCameraPosition;

varying vec3 vNormal;
varying vec3 vWorldPosition;
varying vec2 vUv;

float hash(vec2 p) {
  return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453123);
}

float noise(vec2 p) {
  vec2 i = floor(p);
  vec2 f = fract(p);
  float a = hash(i);
  float b = hash(i + vec2(1.0, 0.0));
  float c = hash(i + vec2(0.0, 1.0));
  float d = hash(i + vec2(1.0, 1.0));
  vec2 u = f * f * (3.0 - 2.0 * f);
  return mix(a, b, u.x) + (c - a) * u.y * (1.0 - u.x) + (d - b) * u.x * u.y;
}

void main() {
  vec3 normal = normalize(vNormal);
  vec3 viewDir = normalize(uCameraPosition - vWorldPosition);

  float fresnel = pow(1.0 - max(dot(viewDir, normal), 0.0), 3.2);
  float softFresnel = pow(1.0 - max(dot(viewDir, normal), 0.0), 1.8);

  vec2 gridUv = vUv * vec2(42.0, 24.0);
  float lineX = smoothstep(0.04, 0.0, abs(fract(gridUv.x) - 0.5));
  float lineY = smoothstep(0.05, 0.0, abs(fract(gridUv.y) - 0.5));
  float techLines = max(lineX, lineY) * 0.055;

  float surfaceNoise = noise(vUv * 18.0 + uTime * 0.01) * 0.045;
  float microNoise = noise(vUv * 64.0) * 0.03;

  float sparkSeed = hash(floor(vUv * vec2(96.0, 56.0)));
  float sparks = step(0.985, sparkSeed) * (0.45 + 0.55 * sin(uTime * 1.4 + sparkSeed * 40.0));

  vec3 color = mix(uBaseColor, uSurfaceColor, 0.35 + surfaceNoise + microNoise);
  color += uGlowColor * techLines;
  color += uHighlight * sparks * 0.55;
  color += uGlowColor * softFresnel * 0.35;
  color += uHighlight * fresnel * 1.05;

  // Keep silhouette readable against page background
  color = max(color, uBaseColor * 1.25);

  gl_FragColor = vec4(color, 1.0);
}
`;

export const atmosphereVertexShader = /* glsl */ `
varying vec3 vNormal;
varying vec3 vWorldPosition;

void main() {
  vec4 worldPosition = modelMatrix * vec4(position, 1.0);
  vWorldPosition = worldPosition.xyz;
  vNormal = normalize(mat3(modelMatrix) * normal);
  gl_Position = projectionMatrix * viewMatrix * worldPosition;
}
`;

export const atmosphereFragmentShader = /* glsl */ `
uniform vec3 uGlowColor;
uniform vec3 uHighlight;
uniform vec3 uCameraPosition;

varying vec3 vNormal;
varying vec3 vWorldPosition;

void main() {
  vec3 normal = normalize(vNormal);
  vec3 viewDir = normalize(uCameraPosition - vWorldPosition);
  float fresnel = pow(1.0 - max(dot(viewDir, normal), 0.0), 2.8);
  float alpha = fresnel * 0.28;
  vec3 color = mix(uGlowColor, uHighlight, fresnel * 0.45);
  gl_FragColor = vec4(color, alpha);
}
`;
