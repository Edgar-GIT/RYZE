export const planetVertexShader = /* glsl */ `
varying vec3 vNormal;
varying vec3 vWorldPosition;
varying vec3 vLocalPosition;
varying vec2 vUv;

float hash(vec2 p) {
  return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453123);
}

void main() {
  vUv = uv;
  vLocalPosition = position;

  vec2 tiles = floor(uv * vec2(52.0, 30.0));
  float panel = hash(tiles);
  float height = step(0.42, panel) * (0.35 + 0.65 * hash(tiles + 19.7));
  vec3 displaced = position + normal * height * 0.045;

  vec4 worldPosition = modelMatrix * vec4(displaced, 1.0);
  vWorldPosition = worldPosition.xyz;
  vNormal = normalize(mat3(modelMatrix) * normal);
  gl_Position = projectionMatrix * viewMatrix * worldPosition;
}
`;

export const planetFragmentShader = /* glsl */ `
uniform float uTime;
uniform vec4 uMeteorActive;
uniform vec4 uMeteorProgress;
uniform vec4 uMeteorOffset;
uniform vec3 uBaseColor;
uniform vec3 uSurfaceColor;
uniform vec3 uGlowColor;
uniform vec3 uHighlight;
uniform vec3 uGoldColor;
uniform vec3 uCameraPosition;

varying vec3 vNormal;
varying vec3 vWorldPosition;
varying vec3 vLocalPosition;
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

float meteorAt(float active, float progress, float offset) {
  if (active < 0.01) {
    return 0.0;
  }

  vec2 dir = normalize(vec2(1.0, 0.22 + offset * 0.55));
  vec2 start = vec2(-0.15, mix(0.08, 0.78, offset));
  vec2 head = start + dir * (progress * 1.45);
  vec2 toPixel = vUv - head;
  float along = -dot(toPixel, dir);
  float across = abs(dot(toPixel, vec2(-dir.y, dir.x)));
  float headGlow = exp(-dot(toPixel, toPixel) * 1100.0);
  float trail = smoothstep(0.48, 0.0, along)
    * smoothstep(0.018, 0.0, across)
    * (1.0 - smoothstep(0.0, 0.48, along));
  float life = smoothstep(0.0, 0.06, progress) * (1.0 - smoothstep(0.9, 1.0, progress));
  return (headGlow * 3.0 + trail * 1.7) * life * active;
}

void main() {
  vec3 normal = normalize(vNormal);
  vec3 viewDir = normalize(uCameraPosition - vWorldPosition);

  float fresnel = pow(1.0 - max(dot(viewDir, normal), 0.0), 3.0);
  float softFresnel = pow(1.0 - max(dot(viewDir, normal), 0.0), 1.7);

  // Panel / greeble surface with deep groove shadows
  vec2 tileScale = vec2(52.0, 30.0);
  vec2 tiles = floor(vUv * tileScale);
  vec2 f = fract(vUv * tileScale);
  float panel = hash(tiles);
  float panelB = hash(tiles + 7.1);
  float heightTone = mix(0.55, 1.0, step(0.42, panel) * (0.4 + 0.6 * panelB));

  float edge = min(min(f.x, 1.0 - f.x), min(f.y, 1.0 - f.y));
  float groove = smoothstep(0.0, 0.07, edge);
  float creviceShadow = mix(0.12, 1.0, groove);

  float micro = noise(vUv * 90.0) * 0.08;
  float panelGlow = step(0.935, panel) * (0.25 + 0.35 * abs(sin(uTime * 0.7 + panel * 20.0)));

  // Directional self-shadowing for volume
  vec3 keyLight = normalize(vec3(0.55, 0.85, 0.45));
  vec3 fillLight = normalize(vec3(-0.7, -0.2, 0.4));
  float key = max(dot(normal, keyLight), 0.0);
  float fill = max(dot(normal, fillLight), 0.0) * 0.25;
  float lighting = 0.22 + key * 0.85 + fill;
  float rimLit = fresnel * (0.55 + 0.45 * max(dot(normal, normalize(vec3(-0.8, 0.2, 0.4))), 0.0));

  float meteors =
    meteorAt(uMeteorActive.x, uMeteorProgress.x, uMeteorOffset.x) +
    meteorAt(uMeteorActive.y, uMeteorProgress.y, uMeteorOffset.y) +
    meteorAt(uMeteorActive.z, uMeteorProgress.z, uMeteorOffset.z) +
    meteorAt(uMeteorActive.w, uMeteorProgress.w, uMeteorOffset.w);

  vec3 color = mix(uBaseColor, uSurfaceColor, 0.4 + micro);
  color *= heightTone * creviceShadow * lighting;
  color += uGlowColor * panelGlow;
  color += uGlowColor * softFresnel * 0.22;
  color += uHighlight * rimLit * 0.95;
  color += uGoldColor * meteors * 2.6;

  // Soft contact occlusion near lower hemisphere
  float underside = smoothstep(0.15, -0.55, normalize(vLocalPosition).y);
  color *= mix(1.0, 0.55, underside * 0.65);

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
  float fresnel = pow(1.0 - max(dot(viewDir, normal), 0.0), 2.4);
  float alpha = fresnel * 0.42;
  vec3 color = mix(uGlowColor, uHighlight, fresnel * 0.55);
  gl_FragColor = vec4(color, alpha);
}
`;
