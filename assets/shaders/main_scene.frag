#version 410 core

in vec3 vWorldPos;
in vec3 vNormal;
in vec3 vViewPos;

uniform vec3 uSunDir;
uniform vec3 uViewPos;
uniform mat4 uLightViewProj[4];
uniform float uCascadeFarPlanes[4];
uniform int  uCascadeCount;
uniform sampler2DShadow uShadowMaps[4];
uniform bool uDebugMode;
uniform vec3 uObjectColor;
uniform float uShadowMapResolution;
uniform int   uPCFSamples;
uniform float uPCFKernelRadii[4];

out vec4 FragColor;

uint pcgHash(uvec2 v) {
    uint state = v.x * 747796405u + v.y * 2891336453u;
    uint word = ((state >> ((state >> 28u) + 4u)) ^ state) * 277803737u;
    return (word >> 22u) ^ word;
}

float randAngle(vec2 fragCoord) {
    return float(pcgHash(uvec2(fragCoord))) / float(0xFFFFFFFFu) * 6.28318530718;
}

const float GOLDEN_ANGLE = 2.399963229728653;

vec2 vogelSample(int i, int N) {
    float r = sqrt(float(i) + 0.5) / sqrt(float(N));
    float theta = float(i) * GOLDEN_ANGLE;
    return r * vec2(cos(theta), sin(theta));
}

float calcShadowPCF(int cascadeIdx, vec4 lightSpacePos, vec2 texelSize, float bias) {
    vec3 projCoords = lightSpacePos.xyz / lightSpacePos.w;
    projCoords = projCoords * 0.5 + 0.5;

    if (projCoords.z > 1.0) {
        return 1.0;
    }
    projCoords.z -= bias;

    float angle = randAngle(gl_FragCoord.xy);
    vec2 rot = vec2(cos(angle), sin(angle));

    float kernelRadius = uPCFKernelRadii[cascadeIdx];
    float shadow = 0.0;
    for (int i = 0; i < uPCFSamples; i++) {
        vec2 samplePos = vogelSample(i, uPCFSamples);
        vec2 sampleOffset = vec2(
            samplePos.x * rot.x - samplePos.y * rot.y,
            samplePos.x * rot.y + samplePos.y * rot.x
        );

        vec2 offset = sampleOffset * kernelRadius * texelSize;
        shadow += texture(uShadowMaps[cascadeIdx],
            vec3(projCoords.xy + offset, projCoords.z));
    }

    return shadow / float(uPCFSamples);
}

const float CASCADE_BLEND_ZONE = 0.15;

int selectCascade(float viewDepth) {
    for (int i = 0; i < uCascadeCount; i++) {
        if (viewDepth < uCascadeFarPlanes[i]) {
            return i;
        }
    }
    return uCascadeCount - 1;
}

void main() {
    vec3 N = normalize(vNormal);
    float NdotL = dot(N, -uSunDir);
    float wrappedNdotL = NdotL * 0.5 + 0.5;

    float diffuseLight = wrappedNdotL * 0.7 + max(NdotL, 0.0) * 0.3;
    vec3 ambient = vec3(0.18, 0.18, 0.25);
    vec3 diffuse = uObjectColor * diffuseLight;

    float viewDepth = -vViewPos.z;
    int cascadeIdx = uCascadeCount - 1;
    float cascadeBlend = 0.0;

    for (int i = 0; i < uCascadeCount; i++) {
        if (viewDepth < uCascadeFarPlanes[i]) {
            cascadeIdx = i;
            if (i < uCascadeCount - 1) {
                float blendStart = uCascadeFarPlanes[i] * (1.0 - CASCADE_BLEND_ZONE);
                if (viewDepth > blendStart) {
                    cascadeBlend = smoothstep(blendStart, uCascadeFarPlanes[i], viewDepth);
                }
            }
            break;
        }
    }

    float NdotL_clamped = max(NdotL, 0.0);
    float bias = max(0.0008 * (1.0 - NdotL_clamped), 0.0001);

    vec2 texelSize = 1.0 / vec2(uShadowMapResolution);
    vec4 lightSpacePos = uLightViewProj[cascadeIdx] * vec4(vWorldPos, 1.0);
    float shadow = calcShadowPCF(cascadeIdx, lightSpacePos, texelSize, bias);
    if (cascadeBlend > 0.0) {
        vec4 lightSpacePos2 = uLightViewProj[cascadeIdx + 1] * vec4(vWorldPos, 1.0);
        float shadowNext = calcShadowPCF(cascadeIdx + 1, lightSpacePos2, texelSize, bias);
        shadow = mix(shadow, shadowNext, cascadeBlend);
    }

    vec3 color = ambient + diffuse * shadow;

    if (uDebugMode) {
        vec3 cascadeColors[4] = vec3[4](
            vec3(1.0, 0.2, 0.2),
            vec3(0.2, 1.0, 0.2),
            vec3(0.2, 0.4, 1.0),
            vec3(1.0, 1.0, 0.2)
        );
        
        vec3 debugColor = (cascadeBlend > 0.0)
            ? mix(cascadeColors[cascadeIdx], cascadeColors[cascadeIdx + 1], cascadeBlend)
            : cascadeColors[cascadeIdx];

        color = mix(color, debugColor, 0.35);
    }

    FragColor = vec4(color, 1.0);
}
