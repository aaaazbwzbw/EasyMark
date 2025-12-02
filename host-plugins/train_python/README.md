# YOLOv8/YOLOv11 训练插件

EasyMark 官方训练插件，基于 Ultralytics 框架实现。

## 功能特性

- ✅ 目标检测 (Detection)
- ✅ 姿态估计 (Pose)
- ✅ 实例分割 (Segmentation)

## 支持的模型

### YOLOv8 系列
- YOLOv8n/s/m/l/x (检测)
- YOLOv8n/s/m/l/x-pose (姿态)
- YOLOv8n/s/m/l/x-seg (分割)

### YOLOv11 系列
- YOLOv11n/s/m/l/x (检测)
- YOLOv11n/s/m/l/x-pose (姿态)
- YOLOv11n/s/m/l/x-seg (分割)

## 依赖

```
ultralytics>=8.0.0
torch>=2.0.0
torchvision>=0.15.0
numpy>=1.20.0
opencv-python>=4.6.0
pillow>=9.0.0
pyyaml>=6.0
tqdm>=4.60.0
```

## 使用方式

此插件由 EasyMark 宿主程序自动调用，不建议手动运行。

### 手动测试

```bash
python main.py --socket-port=12345 --task-id=test_001
```

## 开发说明

详见 [训练插件开发规范](../../docs/training-plugin-dev-guide.md)
