# Common Dataset Import Plugin

EasyMark 通用数据集导入插件，支持多种常见数据集格式。

## 支持的格式

| 格式 | 说明 | 标注类型 |
|------|------|----------|
| **COCO** | Microsoft COCO 格式 | 矩形框、关键点 |
| **YOLO** | YOLO 文本格式 | 矩形框 |
| **Pascal VOC** | Pascal VOC XML 格式 | 矩形框 |

## 自动格式识别

插件会自动扫描目录结构并识别数据集格式：

- **COCO**: 查找包含 `images`、`annotations`、`categories` 的 JSON 文件
- **YOLO**: 查找 `labels` 目录下的 `.txt` 标注文件
- **VOC**: 查找 `Annotations` 目录下的 `.xml` 标注文件

## 数据集结构示例

### COCO 格式

```
dataset/
├── annotations/
│   └── instances_train.json
└── images/
    ├── 000001.jpg
    └── ...
```

### YOLO 格式

```
dataset/
├── classes.txt
├── images/
│   ├── 000001.jpg
│   └── ...
└── labels/
    ├── 000001.txt
    └── ...
```

### Pascal VOC 格式

```
dataset/
├── Annotations/
│   ├── 000001.xml
│   └── ...
└── JPEGImages/
    ├── 000001.jpg
    └── ...
```

## 构建

```bash
cd host-plugin/dataset-common

# Windows
go build -o common-importer.exe .

# Linux / macOS
go build -o common-importer .
```

## 命令行测试

```bash
# 探测格式
echo '{"rootPath": "D:/datasets/my-dataset"}' | common-importer.exe detect

# 导入数据集
echo '{"rootPath": "D:/datasets/my-dataset", "formatId": "dataset.common:coco", "params": {}}' | common-importer.exe import
```

## 参数说明

| 参数 | 说明 |
|------|------|
| `format` | 强制指定格式：`coco`、`yolo`、`voc`，留空则自动识别 |
| `annotationFile` | COCO: JSON 文件路径；VOC: XML 目录路径 |
| `imagesDir` | 图片目录路径 |
| `classesFile` | YOLO 类别文件路径 |

## 许可证

MIT License
