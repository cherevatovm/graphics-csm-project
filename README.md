# Cascaded Shadow Maps

Динамические тени для открытого мира с каскадными теневыми картами (CSM) на **Go + OpenGL 4.1**.

---

## Функциональность

### Cascaded Shadow Maps (4 каскада)
Frustum камеры разбивается на 4 сегмента по **Practical Split Scheme** (λ = 0.75). Для каждого сегмента строится отдельная теневая карта: ближние каскады получают больше текселей на метр геометрии, дальние - меньше. Матрицы LightViewProj квантуются к размеру текселя (**texel-snapping**) - при движении камеры тени не дрожат. Центр frustum-сегмента и радиус описанной сферы вычисляются через обратную матрицу `(Proj × View)⁻¹`, что даёт точные world-space координаты углов.

### PCF-фильтрация (Vogel disk, настраиваемое количество сэмплов)
Сэмплы распределяются по **Vogel disk** (угол золотого сечения ≈ 137.5°): `r = √(i+0.5)/√N`, `θ = i × 2.4 рад`. В отличие от регулярной сетки, спираль не создаёт структурных паттернов. Количество сэмплов и радиус ядра задаются через uniform-переменные (`uPCFSamples`, `uPCFKernelRadii`) и могут различаться для разных каскадов. На каждый пиксель диск поворачивается на псевдослучайный угол (PCG-хеш из экранных координат).

### Борьба с артефактами теней
| Артефакт | Причина | Решение |
|----------|---------|---------|
| **Shadow acne** | Дискретизация depth-буфера | `glPolygonOffset(2.0, 2000.0)` + slope-based bias в шейдере |
| **Peter-panning** | Смещение "отрывает" тень от объекта | `glCullFace(GL_FRONT)` - рендерятся задние грани, чьи артефакты не видны |
| **Shimmering** | Субтексельное смещение матрицы каждый кадр | Texel-snapping: квантование `lvp[12]` и `lvp[13]` к границам текселей |
| **Сетчатый шум PCF** | Регулярная сетка сэмплов | Vogel disk + случайный поворот на пиксель |
| **Тонкая геометрия** | Передняя и задняя грани в одном текселе | `GL_LEQUAL` (равные глубины → светло) + увеличенный polyOffset |

### Загрузка моделей OBJ
Парсер Wavefront .obj поддерживает форматы граней `v/vt/vn`, `v//vn` и `v`. Четырёхугольники автоматически триангулируются: `(a,b,c,d) → (a,b,c) + (a,c,d)`. Раздельные индексы позиций/нормалей (OBJ) разворачиваются в единый индексный буфер (OpenGL) через HashMap уникальных комбинаций. Если нормали отсутствуют, они вычисляются автоматически усреднением нормалей прилегающих граней.

### Дебаг-визуализация (F1)
**Цветовое тонирование**: каждый фрагмент подмешивает цвет своего каскада - красный (C0, ~8 м), зелёный (C1, ~45 м), синий (C2, ~200 м), жёлтый (C3, ~400 м)

### Инстансинг
Каждая OBJ-модель загружается один раз (VAO/VBO/EBO). Экземпляры создаются как отдельные `Mesh` с общей геометрией, но индивидуальными `ModelMatrix` (позиция, поворот, масштаб) и `Color`. Это снижает потребление VRAM и ускоряет рендеринг.

---

## Управление

| Клавиша | Действие |
|---------|----------|
| **W A S D** | Перемещение камеры (вперёд/влево/назад/вправо) |
| **Мышь** | Вращение камеры (yaw/pitch) |
| **Space / Shift** | Камера вверх / вниз |
| **← ↑ ↓ →** | Вращение солнца (азимут / возвышение) |
| **F1** | Дебаг-режим: цветовое тонирование каскадов + frustum-линии |
| **ESC** | Выход |

---

## Архитектура

```
graphics-csm-project/
├── cmd/main.go                          # Точка входа: GLFW, главный цикл
├── assets/
│   ├── shaders/
│   │   ├── main_scene.vert/.frag        # Основной шейдер (PCF + half-Lambert)
│   │   ├── shadow_depth.vert/.frag      # Шейдер теневого прохода
│   │   └── debug_lines.vert/.frag       # Шейдер отладочных frustum-линий
│   └── 3d_models
├── pkg/
│   ├── shader/loader.go                 # Компиляция GLSL из файлов
│   ├── camera/camera.go                 # Free-fly камера (Эйлер: yaw/pitch)
│   ├── shadow/
│   │   ├── fbo.go                       # FBO-менеджер: текстуры глубин
│   │   ├── cascade.go                   # Расчёт каскадов (Practical Split Scheme)
│   │   └── pcf.go                       # Константы PCF, параметры смещения
│   ├── scene/
│   │   ├── scene.go                     # Mesh, Scene, фабрики (Cube, Cone)
│   │   ├── terrain.go                   # Процедурный ландшафт
│   │   └── objloader.go                 # Парсер Wavefront .obj
│   └── renderer/
│       ├── shadow_renderer.go           # Теневой проход (CULL_FRONT + polyOffset)
│       ├── renderer.go                  # Основной проход с CSM
│       └── debug.go                     # Оверлей: frustum-линии каскадов
├── go.mod / go.sum
└── README.md
```

### Поток данных одного кадра

```
┌─────────────────────────────────────────────────────────┐
│ PHASE 1: SHADOW PASS                                    │
│  camera + sunDir → CascadeCalculator → LightViewProj[4] │
│  FOR EACH cascade:                                      │
│    bind FBO → clear depth → draw scene (CULL_FRONT)     │
│  OUTPUT: DepthMaps[4] + LightViewProj[4]                │
├─────────────────────────────────────────────────────────┤
│ PHASE 2: MAIN PASS                                      │
│  bind default FBO → clear color+depth                   │
│  FOR EACH mesh:                                         │
│    selectCascade(viewDepth)                             │
│    PCF(DepthMap[cascade], lightSpacePos, bias)          │
│    color = ambient + objectColor * halfLambert * shadow │
│  OUTPUT: final framebuffer                              │
├─────────────────────────────────────────────────────────┤
│ PHASE 3: DEBUG (F1)                                     │
│  draw frustum wireframes + cascade color tint           │
└─────────────────────────────────────────────────────────┘
```

---

## Технические детали CSM

### Разбиение frustum (Practical Split Scheme)

$$C_i = \lambda \cdot n \cdot \left(\frac{f}{n}\right)^{\frac{i}{N}} + (1 - \lambda) \cdot \left(n + \frac{i}{N}(f - n)\right)$$

| Параметр | Значение |
|----------|----------|
| Каскадов (N) | 4 |
| λ (гибрид) | 0.75 |
| Near/Far камеры | 0.1 / 400 м |
| Разрешение карты | 2048×2048 |

| Каскад | Far plane |
|--------|-----------|
| C0 (красный) | ~8 м |
| C1 (зелёный) | ~45 м |
| C2 (синий) | ~200 м |
| C3 (жёлтый) | ~400 м |

### PCF (Percentage-Closer Filtering)

- **Vogel disk** с настраиваемым числом сэмплов и радиусом ядра на каскад
- Случайный поворот диска на каждый пиксель
- Аппаратное сравнение глубины: `sampler2DShadow` + `GL_LEQUAL`
- Slope-based depth bias: предотвращает самозатенение

### Борьба с артефактами

| Артефакт | Решение |
|----------|---------|
| Shadow acne | `glPolygonOffset(2.0, 2000.0)` + slope bias |
| Peter-panning | `glCullFace(GL_FRONT)` на теневом проходе |
| Shimmering | Texel-snapping LightViewProj |
| Паттерны сетки | Vogel disk вместо регулярной сетки |
| Тонкая геометрия | `GL_LEQUAL` + увеличенный polyOffset |

---

## Запуск

### Требования

- **Go** 1.21+
- **OpenGL** 4.1+ (Core Profile)
- **GLFW** 3.3+

### Сборка и запуск

```bash
git clone https://github.com/cherevatovm/graphics-csm-project
cd graphics-csm-project
go mod tidy
go run ./cmd
```

### Зависимости

| Пакет | Назначение |
|-------|-----------|
| `github.com/go-gl/gl` | OpenGL биндинги |
| `github.com/go-gl/glfw/v3.3/glfw` | GLFW (окно, ввод) |
| `github.com/go-gl/mathgl/mgl32` | Линейная алгебра (Vec3, Mat4) |

---

## Техника
- 3–4 каскада теней (split by distance), визуализация каскадов разными цветами для
отладки
- Расширение классического Shadow Mapping - добавляется цикл рендера в
несколько FBO
- Мягкие тени благодаря PCF фильтрации
