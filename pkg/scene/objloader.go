package scene

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/go-gl/mathgl/mgl32"
)

type faceVertex struct {
	posIdx  int
	normIdx int
}

func LoadOBJ(path string) (*Mesh, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("открытие %s: %w", path, err)
	}
	defer file.Close()

	var positions []mgl32.Vec3
	var normals []mgl32.Vec3
	var faces [][]faceVertex

	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "v":
			if len(parts) < 4 {
				continue
			}
			x, _ := strconv.ParseFloat(parts[1], 32)
			y, _ := strconv.ParseFloat(parts[2], 32)
			z, _ := strconv.ParseFloat(parts[3], 32)
			positions = append(positions, mgl32.Vec3{float32(x), float32(y), float32(z)})

		case "vn":
			if len(parts) < 4 {
				continue
			}
			nx, _ := strconv.ParseFloat(parts[1], 32)
			ny, _ := strconv.ParseFloat(parts[2], 32)
			nz, _ := strconv.ParseFloat(parts[3], 32)
			normals = append(normals, mgl32.Vec3{float32(nx), float32(ny), float32(nz)})

		case "f":
			faceVerts := make([]faceVertex, 0, len(parts)-1)
			for _, tok := range parts[1:] {
				fv := parseFaceVertex(tok)
				faceVerts = append(faceVerts, fv)
			}

			if len(faceVerts) < 3 {
				continue
			}

			if len(faceVerts) == 4 {
				faces = append(faces, []faceVertex{faceVerts[0], faceVerts[1], faceVerts[2]})
				faces = append(faces, []faceVertex{faceVerts[0], faceVerts[2], faceVerts[3]})
			} else if len(faceVerts) == 3 {
				faces = append(faces, faceVerts)
			} else {
				for j := 1; j < len(faceVerts)-1; j++ {
					faces = append(faces, []faceVertex{faceVerts[0], faceVerts[j], faceVerts[j+1]})
				}
			}

		default:
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("чтение %s (строка %d): %w", path, lineNo, err)
	}
	if len(positions) == 0 {
		return nil, fmt.Errorf("файл %s не содержит вершин", path)
	}
	if len(normals) == 0 {
		normals = computeNormals(positions, faces)
	}

	unrolled := make(map[uint64]uint32)
	var interleaved []float32
	var indices []uint32

	for _, tri := range faces {
		for _, fv := range tri {
			pi := fv.posIdx - 1
			ni := fv.normIdx - 1
			if ni < 0 {
				ni = 0
			}
			if pi < 0 || pi >= len(positions) {
				pi = 0
			}
			if ni >= len(normals) {
				ni = 0
			}

			key := (uint64(pi) << 20) | uint64(ni)
			idx, exists := unrolled[key]
			if !exists {
				p := positions[pi]
				n := normals[ni]
				interleaved = append(interleaved,
					p.X(), p.Y(), p.Z(),
					n.X(), n.Y(), n.Z(),
				)
				idx = uint32(len(interleaved)/6 - 1)
				unrolled[key] = idx
			}

			indices = append(indices, idx)
		}
	}

	return createMesh(interleaved, indices), nil
}

func parseFaceVertex(tok string) faceVertex {
	var fv faceVertex

	slash1 := strings.IndexByte(tok, '/')
	if slash1 < 0 {
		fv.posIdx, _ = strconv.Atoi(tok)
		return fv
	}

	fv.posIdx, _ = strconv.Atoi(tok[:slash1])
	rest := tok[slash1+1:]
	if len(rest) == 0 {
		return fv
	}

	slash2 := strings.IndexByte(rest, '/')
	if slash2 < 0 {
		return fv
	}

	normStr := rest[slash2+1:]
	if normStr != "" {
		fv.normIdx, _ = strconv.Atoi(normStr)
	}

	return fv
}

func computeNormals(positions []mgl32.Vec3, faces [][]faceVertex) []mgl32.Vec3 {
	normals := make([]mgl32.Vec3, len(positions))

	for _, tri := range faces {
		if len(tri) < 3 {
			continue
		}

		i0 := tri[0].posIdx - 1
		i1 := tri[1].posIdx - 1
		i2 := tri[2].posIdx - 1
		if i0 < 0 || i1 < 0 || i2 < 0 ||
			i0 >= len(positions) || i1 >= len(positions) || i2 >= len(positions) {
			continue
		}

		p0, p1, p2 := positions[i0], positions[i1], positions[i2]
		edge1 := p1.Sub(p0)
		edge2 := p2.Sub(p0)
		faceNormal := edge1.Cross(edge2)

		normals[i0] = normals[i0].Add(faceNormal)
		normals[i1] = normals[i1].Add(faceNormal)
		normals[i2] = normals[i2].Add(faceNormal)
	}

	for i := range normals {
		if normals[i].Len() > 1e-9 {
			normals[i] = normals[i].Normalize()
		} else {
			normals[i] = mgl32.Vec3{0, 1, 0}
		}
	}

	return normals
}
