# tin

三角网生成等高线

## 开发语言

golang

## 导入

```golang
go get github.com/xhymf1992/tin
```

## 使用案例

```golang
import "github/xhymf1992/tin"

func main() {
    // 从stl文件读取三角网，也可以按*[]tin.Triangle自行构建
    path := "path\xxx.stl"
    triangles, err := tin.ReadStl(path)
    if err != nil {
        fmt.Println(err)
        return
    }
    
    // 使用tin.Tin构建tin实例
    tin := tin.Tin{
        Triangles: triangles, // 传入*[]tin.Triangle
    }
    
    // 先初始化（内部有一些处理）
    tin.Init()

    // 调用接口，传入高程间隔，单位米，并生成等高线
    contourSet := tin.GenerateContour(5) // 每隔5米生成等高线
    fmt.Println(contourSet) // contourSet为等高线数据，结构为 *[][][]tin.Vector3
}
```