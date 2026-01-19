package tin

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
)

type Vector3 struct {
	x float64
	y float64
	z float64
}

type Triangle struct {
	pt0 Vector3
	pt1 Vector3
	pt2 Vector3
}

type Box struct {
	min Vector3
	max Vector3
}

type Tin struct {
	Triangles        *[]Triangle
	Box              Box
	lines            [][]Vector3
	linesBelongtoTri [][]int
	initialized      bool
}

func (tin *Tin) Init() {
	tin.Box = Box{
		min: Vector3{
			x: 1e9,
			y: 1e9,
			z: 1e9,
		},
		max: Vector3{
			x: -1e9,
			y: -1e9,
			z: -1e9,
		},
	}
	tin.lines = [][]Vector3{}
	tin.linesBelongtoTri = [][]int{}
	tin.initialized = false

	tin.initBox()
}

func (tin *Tin) initBox() {
	for _, tri := range *tin.Triangles {
		tin.Box.min.x = min(tri.pt0.x, tri.pt1.x, tri.pt2.x, tin.Box.min.x)
		tin.Box.min.y = min(tri.pt0.y, tri.pt1.y, tri.pt2.y, tin.Box.min.y)
		tin.Box.min.z = min(tri.pt0.z, tri.pt1.z, tri.pt2.z, tin.Box.min.z)

		tin.Box.max.x = max(tri.pt0.x, tri.pt1.x, tri.pt2.x, tin.Box.max.x)
		tin.Box.max.y = max(tri.pt0.y, tri.pt1.y, tri.pt2.y, tin.Box.max.y)
		tin.Box.max.z = max(tri.pt0.z, tri.pt1.z, tri.pt2.z, tin.Box.max.z)
	}
}

func (tin *Tin) calLineInfo() {
	lineMap := make(map[string]int, 0)
	nextLineIdx := 0
	for idxTri, tri := range *tin.Triangles {
		triPt0Str := fmt.Sprintf("%f_%f_%f", tri.pt0.x, tri.pt0.y, tri.pt0.z)
		triPt1Str := fmt.Sprintf("%f_%f_%f", tri.pt1.x, tri.pt1.y, tri.pt1.z)
		triPt2Str := fmt.Sprintf("%f_%f_%f", tri.pt2.x, tri.pt2.y, tri.pt2.z)

		line1 := fmt.Sprintf("%s_%s", triPt1Str, triPt0Str)
		line1Idx, ok := lineMap[line1]
		if ok {
			tin.linesBelongtoTri[line1Idx] = append(tin.linesBelongtoTri[line1Idx], idxTri)
			delete(lineMap, line1)
		} else {
			tin.lines = append(tin.lines, []Vector3{
				tri.pt0,
				tri.pt1,
			})
			tin.linesBelongtoTri = append(tin.linesBelongtoTri, []int{
				idxTri,
			})
			lineMap[fmt.Sprintf("%s_%s", triPt0Str, triPt1Str)] = nextLineIdx
			nextLineIdx++
		}

		line2 := fmt.Sprintf("%s_%s", triPt2Str, triPt1Str)
		line2Idx, ok := lineMap[line2]
		if ok {
			tin.linesBelongtoTri[line2Idx] = append(tin.linesBelongtoTri[line2Idx], idxTri)
			delete(lineMap, line2)
		} else {
			tin.lines = append(tin.lines, []Vector3{
				tri.pt1,
				tri.pt2,
			})
			tin.linesBelongtoTri = append(tin.linesBelongtoTri, []int{
				idxTri,
			})
			lineMap[fmt.Sprintf("%s_%s", triPt1Str, triPt2Str)] = nextLineIdx
			nextLineIdx++
		}

		line3 := fmt.Sprintf("%s_%s", triPt0Str, triPt2Str)
		line3Idx, ok := lineMap[line3]
		if ok {
			tin.linesBelongtoTri[line3Idx] = append(tin.linesBelongtoTri[line3Idx], idxTri)
			delete(lineMap, line3)
		} else {
			tin.lines = append(tin.lines, []Vector3{
				tri.pt2,
				tri.pt0,
			})
			tin.linesBelongtoTri = append(tin.linesBelongtoTri, []int{
				idxTri,
			})
			lineMap[fmt.Sprintf("%s_%s", triPt2Str, triPt0Str)] = nextLineIdx
			nextLineIdx++
		}
	}
}

func (tin *Tin) getZValueSetByInterval(zMin float64, zMax float64, zInterval float64) []float64 {
	zSet := make([]float64, 0, 1)
	if zInterval > 0 {
		zIntervalNew := math.Round(zInterval)

		zInput := math.Ceil(zMin/zIntervalNew) * zIntervalNew
		for true {
			if zInput > zMax {
				break
			}

			zSet = append(zSet, zInput)
			zInput += zIntervalNew
		}
	}

	return zSet
}

var count = 0

func (tin *Tin) getNextPt(start int, usesPt []bool, ptsBelongtoLine []int, contour *[]int) {
	count++
	*contour = append(*contour, start)
	usesPt[start] = true

	for _, curTriIdx := range tin.linesBelongtoTri[ptsBelongtoLine[start]] {
		for idxPt, use := range usesPt {
			if use {
				continue
			}

			if slices.Contains(tin.linesBelongtoTri[ptsBelongtoLine[idxPt]], curTriIdx) {
				tin.getNextPt(idxPt, usesPt, ptsBelongtoLine, contour)
				return
			}
		}
	}
}

func (tin *Tin) getInterPtFromLineByZPlane(z float64) ([]Vector3, []int) {
	pts := make([]Vector3, 0)
	ptsBelongtoLine := make([]int, 0)
	for idxLine, line := range tin.lines {
		var pt Vector3
		exist := false
		if line[0].z <= line[1].z {
			pt, exist = tin.interByZ(line[0], line[1], z)
		} else {
			pt, exist = tin.interByZ(line[1], line[0], z)
		}

		if exist {
			pts = append(pts, pt)
			ptsBelongtoLine = append(ptsBelongtoLine, idxLine)
		}
	}

	return pts, ptsBelongtoLine
}

func (tin *Tin) interByZ(ptMin Vector3, ptMax Vector3, z float64) (Vector3, bool) {
	if ptMin.z != ptMax.z && z >= ptMin.z && z <= ptMax.z {
		rate := (z - ptMin.z) / (ptMax.z - ptMin.z)
		x := ptMin.x + rate*(ptMax.x-ptMin.x)
		y := ptMin.y + rate*(ptMax.y-ptMin.y)
		return Vector3{x: x, y: y, z: z}, true
	}

	return Vector3{}, false
}

func (tin *Tin) GenerateContour(zInterval float64) *[][][]Vector3 {
	// startTime := time.Now().UnixMilli()
	if !tin.initialized {
		tin.calLineInfo()
	}
	// endTime := time.Now().UnixMilli()
	// fmt.Println(endTime - startTime)

	zSet := tin.getZValueSetByInterval(tin.Box.min.z, tin.Box.max.z, zInterval)
	contourSet := make([][][]Vector3, len(zSet))

	for idxZ, z := range zSet {
		pts, ptsBelongtoLine := tin.getInterPtFromLineByZPlane(z)

		usesPt := make([]bool, len(pts)) // 点是否被使用

		contours := make([][]int, 0)

		// 先获取非闭合的
		for true {
			start := -1
			for idx, use := range usesPt {
				if !use && len(tin.linesBelongtoTri[ptsBelongtoLine[idx]]) == 1 {
					start = idx
					break
				}
			}

			if start == -1 {
				break
			}

			contour := make([]int, 0)

			tin.getNextPt(start, usesPt, ptsBelongtoLine, &contour)
			// fmt.Printf("count: %d\n", count)
			count = 0

			contours = append(contours, contour)
		}

		// 再获取闭合的
		for true {
			start := -1
			for idx, use := range usesPt {
				if !use {
					start = idx
					break
				}
			}

			if start == -1 {
				break
			}

			contour := make([]int, 0)

			tin.getNextPt(start, usesPt, ptsBelongtoLine, &contour)
			// fmt.Printf("count: %d\n", count)
			count = 0

			contour = append(contour, contour[0])
			contours = append(contours, contour)
		}

		// 将索引转成坐标
		contoursData := make([][]Vector3, 0)
		for _, line := range contours {
			contourData := make([]Vector3, 0)
			for _, idxPt := range line {
				contourData = append(contourData, pts[idxPt])
			}

			contoursData = append(contoursData, contourData)
		}

		contourSet[idxZ] = contoursData
	}

	return &contourSet
}

func ReadStl(path string) (*[]Triangle, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	triangles := make([]Triangle, 0)
	triangle := Triangle{}
	next_triangle_idx := 0

	reader := bufio.NewReader(file)
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		data := strings.Split(strings.Trim(string(line), " "), " ")
		if len(data) == 0 {
			continue
		}

		switch data[0] {
		case "solid":
			{
			}
		case "facet":
			{
			}
		case "outer":
			{
				triangle = Triangle{}
				next_triangle_idx = 0
			}
		case "vertex":
			{
				pt := Vector3{}

				pt.x, _ = strconv.ParseFloat(data[1], 64)
				pt.y, _ = strconv.ParseFloat(data[2], 64)
				pt.z, _ = strconv.ParseFloat(data[3], 64)

				switch next_triangle_idx {
				case 0:
					triangle.pt0 = pt
				case 1:
					triangle.pt1 = pt
				case 2:
					triangle.pt2 = pt
				}

				next_triangle_idx++
			}
		case "endloop":
			{
				triangles = append(triangles, triangle)
			}
		default:
			{
			}
		}
	}

	return &triangles, nil
}
