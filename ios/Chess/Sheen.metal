#include <metal_stdlib>
#include <SwiftUI/SwiftUI_Metal.h>
using namespace metal;

// A soft light lying on the board. `light` is the light's position in unit
// coordinates of the board (0,0 top-left); it moves as the phone tilts.
[[ stitchable ]] half4 sheen(float2 position, half4 color, float2 size, float2 light, float strength) {
    float2 uv = position / size;
    float d = distance(uv, light);
    float glow = (1.0 - smoothstep(0.0, 0.7, d)) * strength;
    // Lift the colour toward white near the light and dim it slightly far away,
    // which reads as a matte surface rather than a gloss.
    half3 c = color.rgb;
    half3 lit = c + (1.0h - c) * half(glow * 0.32);
    lit *= half(1.0 - 0.07 * smoothstep(0.35, 1.1, d) * strength);
    return half4(lit, color.a);
}
