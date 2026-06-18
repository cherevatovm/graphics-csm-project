#version 410 core

layout(location = 0) in vec3 aPos;
layout(location = 1) in vec3 aNormal;

uniform mat4 uModel;
uniform mat4 uView;
uniform mat4 uProjection;

out vec3 vWorldPos;
out vec3 vNormal;
out vec3 vViewPos;

void main() {
    vec4 worldPos = uModel * vec4(aPos, 1.0);
    vWorldPos = worldPos.xyz;

    vec4 viewPos = uView * worldPos;
    vViewPos = viewPos.xyz;
    vNormal = mat3(uModel) * aNormal;

    gl_Position = uProjection * uView * worldPos;
}
