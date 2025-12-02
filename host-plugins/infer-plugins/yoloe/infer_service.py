#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
EasyMark 推理服务 - YOLOE Visual Prompt 插件
框选一个目标，自动检测图片中所有相似目标
"""
import json
import os
import sys
import traceback
import numpy as np
import cv2

# 全局变量
_model = None
_current_model_path = None
_current_image_path = None
_current_image = None

# 数据根目录和插件目录
DATA_ROOT = os.environ.get("EASYMARK_DATA_PATH", "")
PLUGIN_PATH = os.environ.get("EASYMARK_PLUGIN_PATH", "")


def _send(payload: dict) -> None:
    """发送 JSON 响应"""
    try:
        sys.stdout.write(json.dumps(payload, ensure_ascii=False) + "\n")
        sys.stdout.flush()
    except Exception:
        raise SystemExit(1)


def _log(msg: str) -> None:
    """输出日志到 stderr"""
    try:
        sys.stderr.write(f"[yoloe_service] {msg}\n")
        sys.stderr.flush()
    except Exception:
        pass


def _handle_load_model(request_id: str, weights: str) -> None:
    """加载 YOLOE 模型"""
    global _model, _current_model_path
    
    try:
        model_path = weights
        
        # 优先级：绝对路径 > 插件目录 > 数据目录
        if not os.path.isabs(model_path):
            if PLUGIN_PATH:
                plugin_model = os.path.join(PLUGIN_PATH, weights)
                if os.path.exists(plugin_model):
                    model_path = plugin_model
                    _log(f"Using model from plugin directory: {model_path}")
            
            if not os.path.exists(model_path) and DATA_ROOT:
                data_model = os.path.join(DATA_ROOT, weights)
                if os.path.exists(data_model):
                    model_path = data_model
                    _log(f"Using model from data directory: {model_path}")
        
        if not os.path.exists(model_path):
            _send({
                "requestId": request_id,
                "cmd": "load_model",
                "success": False,
                "error": f"Model file not found: {model_path}"
            })
            return
        
        _log(f"Loading YOLOE model: {model_path}")
        
        # 导入 YOLOE
        from ultralytics import YOLOE
        
        # 加载模型
        _model = YOLOE(model_path)
        _current_model_path = model_path
        
        # 获取设备信息
        device = str(_model.device) if hasattr(_model, 'device') else 'cpu'
        _log(f"Model loaded successfully on device: {device}")
        
        _send({
            "requestId": request_id,
            "cmd": "load_model",
            "success": True,
            "modelPath": model_path,
            "device": device
        })
        
    except Exception as e:
        _log(f"Load model failed: {str(e)}")
        _log(traceback.format_exc())
        _send({
            "requestId": request_id,
            "cmd": "load_model",
            "success": False,
            "error": str(e)
        })


def _handle_set_image(request_id: str, image_path: str) -> None:
    """设置当前图像"""
    global _current_image_path, _current_image
    
    if not _model:
        _send({
            "requestId": request_id,
            "cmd": "set_image",
            "success": False,
            "error": "Model not loaded"
        })
        return
    
    try:
        image_path = image_path.replace('/', os.sep).replace('\\', os.sep)
        if not os.path.isabs(image_path) and DATA_ROOT:
            image_path = os.path.join(DATA_ROOT, image_path)
        
        if not os.path.exists(image_path):
            _send({
                "requestId": request_id,
                "cmd": "set_image",
                "success": False,
                "error": f"Image not found: {image_path}"
            })
            return
        
        _log(f"Setting image: {image_path}")
        
        image = cv2.imread(image_path)
        if image is None:
            raise ValueError(f"Failed to read image: {image_path}")
        
        image_rgb = cv2.cvtColor(image, cv2.COLOR_BGR2RGB)
        
        _current_image_path = image_path
        _current_image = image_rgb
        
        _log("Image set successfully")
        _send({
            "requestId": request_id,
            "cmd": "set_image",
            "success": True,
            "imagePath": image_path,
            "imageSize": {"width": image_rgb.shape[1], "height": image_rgb.shape[0]}
        })
        
    except Exception as e:
        _log(f"Set image failed: {str(e)}")
        _log(traceback.format_exc())
        _send({
            "requestId": request_id,
            "cmd": "set_image",
            "success": False,
            "error": str(e)
        })


def _handle_infer(request_id: str, payload: dict) -> None:
    """执行推理 - 基于框选的视觉提示检测所有相似目标"""
    if not _model:
        _send({
            "requestId": request_id,
            "cmd": "infer",
            "success": False,
            "error": "Model not loaded"
        })
        return
    
    if _current_image is None:
        _send({
            "requestId": request_id,
            "cmd": "infer",
            "success": False,
            "error": "No image set"
        })
        return
    
    try:
        from ultralytics.models.yolo.yoloe import YOLOEVPSegPredictor
        
        img_h, img_w = _current_image.shape[:2]
        conf_threshold = payload.get("conf", 0.3)
        iou_threshold = payload.get("iou", 0.5)
        output_type = payload.get("outputType", "bbox")  # "bbox" 或 "polygon"
        
        # 获取框选区域作为视觉提示
        box = payload.get("box")
        if not box:
            _log("No box provided, returning empty result")
            _send({
                "requestId": request_id,
                "cmd": "infer",
                "success": True,
                "annotations": []
            })
            return
        
        # 转换归一化坐标到像素坐标
        x1 = box["x"] * img_w
        y1 = box["y"] * img_h
        x2 = (box["x"] + box["width"]) * img_w
        y2 = (box["y"] + box["height"]) * img_h
        
        _log(f"Visual prompt box: ({x1:.1f}, {y1:.1f}, {x2:.1f}, {y2:.1f})")
        
        # 构建视觉提示
        visual_prompts = dict(
            bboxes=np.array([[x1, y1, x2, y2]]),
            cls=np.array([0]),  # 单一类别
        )
        
        # 执行推理
        results = _model.predict(
            _current_image_path,
            visual_prompts=visual_prompts,
            predictor=YOLOEVPSegPredictor,
            conf=conf_threshold,
            iou=iou_threshold,
            verbose=False
        )
        
        # 解析结果
        annotations = []
        if results and len(results) > 0:
            result = results[0]
            
            # 获取检测框
            if result.boxes is not None:
                boxes = result.boxes
                for i, box_data in enumerate(boxes):
                    xyxy = box_data.xyxy[0].cpu().numpy()
                    conf = float(box_data.conf[0].cpu().numpy())
                    
                    # 归一化坐标
                    x1_norm = float(xyxy[0]) / img_w
                    y1_norm = float(xyxy[1]) / img_h
                    x2_norm = float(xyxy[2]) / img_w
                    y2_norm = float(xyxy[3]) / img_h
                    
                    if output_type == "polygon" and result.masks is not None and i < len(result.masks):
                        # 返回分割多边形
                        mask = result.masks[i]
                        if hasattr(mask, 'xy') and len(mask.xy) > 0:
                            points = mask.xy[0]
                            if len(points) >= 3:
                                points_normalized = [[float(p[0]) / img_w, float(p[1]) / img_h] for p in points]
                                annotations.append({
                                    "type": "polygon",
                                    "categoryName": "similar",
                                    "confidence": conf,
                                    "data": {
                                        "points": points_normalized
                                    }
                                })
                                continue
                    
                    # 返回矩形框
                    annotations.append({
                        "type": "rect",
                        "categoryName": "similar",
                        "confidence": conf,
                        "data": {
                            "x": x1_norm,
                            "y": y1_norm,
                            "width": x2_norm - x1_norm,
                            "height": y2_norm - y1_norm
                        }
                    })
        
        _log(f"Inference complete: {len(annotations)} detections")
        _send({
            "requestId": request_id,
            "cmd": "infer",
            "success": True,
            "annotations": annotations
        })
        
    except Exception as e:
        _log(f"Inference failed: {str(e)}")
        _log(traceback.format_exc())
        _send({
            "requestId": request_id,
            "cmd": "infer",
            "success": False,
            "error": str(e)
        })


def _handle_unload(request_id: str) -> None:
    """卸载模型"""
    global _model, _current_model_path, _current_image, _current_image_path
    
    _model = None
    _current_model_path = None
    _current_image = None
    _current_image_path = None
    
    _log("Model unloaded")
    _send({
        "requestId": request_id,
        "cmd": "unload",
        "success": True
    })


def _handle_shutdown(request_id: str) -> None:
    """关闭服务"""
    _log("Shutting down")
    _send({
        "requestId": request_id,
        "cmd": "shutdown",
        "success": True
    })
    sys.exit(0)


def main():
    """主循环"""
    _log("YOLOE inference service started (stdio mode)")
    _log(f"DATA_ROOT: {DATA_ROOT}")
    
    try:
        for line in sys.stdin:
            line = line.strip()
            if not line:
                continue
            
            try:
                req = json.loads(line)
                request_id = req.get("requestId", "")
                cmd = req.get("cmd", "")
                
                if cmd == "load_model":
                    _handle_load_model(request_id, req.get("weights", ""))
                elif cmd == "set_image":
                    _handle_set_image(request_id, req.get("path", ""))
                elif cmd == "infer":
                    _handle_infer(request_id, req)
                elif cmd == "unload":
                    _handle_unload(request_id)
                elif cmd == "shutdown":
                    _handle_shutdown(request_id)
                else:
                    _send({
                        "requestId": request_id,
                        "cmd": cmd,
                        "success": False,
                        "error": f"Unknown command: {cmd}"
                    })
            except json.JSONDecodeError as e:
                _log(f"Invalid JSON: {e}")
            except Exception as e:
                _log(f"Error handling request: {e}")
                _log(traceback.format_exc())
    
    except KeyboardInterrupt:
        _log("Interrupted")
    except Exception as e:
        _log(f"Fatal error: {e}")
        _log(traceback.format_exc())
    
    sys.exit(0)


if __name__ == "__main__":
    main()
