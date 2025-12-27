#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
EasyMark 推理服务 - SAM-2 插件
Segment Anything Model 2 交互式分割
"""
import json
import os
import sys
import time
import traceback
from pathlib import Path
import numpy as np
import cv2

# 禁用在线检查
os.environ.setdefault('SAM2_BUILD_ALLOW_ERRORS', '1')

# 全局变量
_predictor = None
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
        sys.stderr.write(f"[sam2_service] {msg}\n")
        sys.stderr.flush()
    except Exception:
        pass


def _handle_load_model(request_id: str, weights: str) -> None:
    """加载 SAM-2 模型"""
    global _predictor, _current_model_path
    
    try:
        # 解析模型路径
        model_path = weights
        
        # 优先级：绝对路径 > 插件目录 > 数据目录
        if not os.path.isabs(model_path):
            # 尝试插件目录
            if PLUGIN_PATH:
                plugin_model = os.path.join(PLUGIN_PATH, weights)
                if os.path.exists(plugin_model):
                    model_path = plugin_model
                    _log(f"Using model from plugin directory: {model_path}")
            
            # 尝试数据目录
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
        
        _log(f"Loading SAM-2 model: {model_path}")
        
        # 导入 SAM-2
        try:
            from sam2.build_sam import build_sam2
            from sam2.sam2_image_predictor import SAM2ImagePredictor
        except ImportError as e:
            _send({
                "requestId": request_id,
                "cmd": "load_model",
                "success": False,
                "error": f"SAM-2 not installed: {str(e)}"
            })
            return
        except OSError as e:
            error_msg = str(e)
            # 检测 VC++ Redistributable 缺失
            if "c10.dll" in error_msg or "WinError 126" in error_msg:
                _send({
                    "requestId": request_id,
                    "cmd": "load_model",
                    "success": False,
                    "error": "Missing Microsoft Visual C++ Redistributable. Please download and install from: https://aka.ms/vs/17/release/vc_redist.x64.exe"
                })
            else:
                _send({
                    "requestId": request_id,
                    "cmd": "load_model",
                    "success": False,
                    "error": f"System error: {error_msg}"
                })
            return
        
        # 根据模型文件自动选择配置
        # 注意：hydra 配置文件在 sam2 包根目录下，不需要路径前缀
        model_file = os.path.basename(model_path)
        
        if "large" in model_file.lower():
            config = "sam2_hiera_l.yaml"
        elif "base_plus" in model_file.lower():
            config = "sam2_hiera_b+.yaml"
        elif "small" in model_file.lower():
            config = "sam2_hiera_s.yaml"
        elif "tiny" in model_file.lower():
            config = "sam2_hiera_t.yaml"
        else:
            config = "sam2_hiera_l.yaml"  # 默认
        
        _log(f"Using config: {config}")
        
        # 确定设备
        import torch
        device = "cuda" if torch.cuda.is_available() else "cpu"
        _log(f"Using device: {device}")
        
        # 加载模型
        sam2_model = build_sam2(config, model_path, device=device)
        _predictor = SAM2ImagePredictor(sam2_model)
        _current_model_path = model_path
        
        _log("Model loaded successfully")
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
    """设置当前图像（SAM-2 需要预处理）"""
    global _current_image_path, _current_image
    
    if not _predictor:
        _send({
            "requestId": request_id,
            "cmd": "set_image",
            "success": False,
            "error": "Model not loaded"
        })
        return
    
    try:
        # 读取图像
        # 统一路径分隔符
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
        
        # 读取图像（RGB）
        image = cv2.imread(image_path)
        if image is None:
            raise ValueError(f"Failed to read image: {image_path}")
        
        image_rgb = cv2.cvtColor(image, cv2.COLOR_BGR2RGB)
        
        # SAM-2 图像预处理
        _predictor.set_image(image_rgb)
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


def _compute_iou(box1: dict, box2: dict) -> float:
    """计算两个边界框的 IoU"""
    # box 格式: {"x": x, "y": y, "width": w, "height": h} (归一化坐标)
    x1 = max(box1["x"], box2["x"])
    y1 = max(box1["y"], box2["y"])
    x2 = min(box1["x"] + box1["width"], box2["x"] + box2["width"])
    y2 = min(box1["y"] + box1["height"], box2["y"] + box2["height"])
    
    if x2 <= x1 or y2 <= y1:
        return 0.0
    
    inter = (x2 - x1) * (y2 - y1)
    area1 = box1["width"] * box1["height"]
    area2 = box2["width"] * box2["height"]
    union = area1 + area2 - inter
    
    return inter / union if union > 0 else 0.0


def _nms_annotations(annotations: list, iou_threshold: float = 0.5) -> list:
    """对标注进行非极大值抑制，去除重叠框"""
    if len(annotations) <= 1:
        return annotations
    
    # 按置信度排序（高到低）
    sorted_anns = sorted(annotations, key=lambda a: a.get("confidence", 0), reverse=True)
    
    keep = []
    while sorted_anns:
        best = sorted_anns.pop(0)
        keep.append(best)
        
        # 计算 best 的边界框
        if best["type"] == "rect":
            best_box = best["data"]
        else:
            # polygon 转 bbox
            points = best["data"]["points"]
            xs = [p[0] for p in points]
            ys = [p[1] for p in points]
            best_box = {
                "x": min(xs), "y": min(ys),
                "width": max(xs) - min(xs), "height": max(ys) - min(ys)
            }
        
        # 过滤掉与 best 重叠过多的框
        remaining = []
        for ann in sorted_anns:
            if ann["type"] == "rect":
                ann_box = ann["data"]
            else:
                points = ann["data"]["points"]
                xs = [p[0] for p in points]
                ys = [p[1] for p in points]
                ann_box = {
                    "x": min(xs), "y": min(ys),
                    "width": max(xs) - min(xs), "height": max(ys) - min(ys)
                }
            
            if _compute_iou(best_box, ann_box) < iou_threshold:
                remaining.append(ann)
        
        sorted_anns = remaining
    
    return keep


def _mask_to_annotation(mask: np.ndarray, score: float, img_w: int, img_h: int, output_type: str = "polygon", max_points: int = 256) -> dict:
    """将掩码转换为标注对象
    
    Args:
        mask: 二值掩码
        score: 置信度
        img_w: 图像宽度
        img_h: 图像高度
        output_type: 输出类型 "polygon" 或 "rect"
        max_points: 多边形最大顶点数，超过会自动简化
    """
    if mask is None or mask.size == 0:
        return None

    is_prob_mask = (mask.dtype != np.uint8) and (float(np.max(mask)) <= 1.0)

    if is_prob_mask:
        mask_prob = mask.astype(np.float32)
        if mask_prob.shape[0] >= 3 and mask_prob.shape[1] >= 3:
            mask_prob = cv2.GaussianBlur(mask_prob, (3, 3), 0)
        mask_uint8 = (mask_prob >= 0.4).astype(np.uint8) * 255
    else:
        mask_uint8 = (mask > 0).astype(np.uint8) * 255

    if mask_uint8.shape[0] >= 3 and mask_uint8.shape[1] >= 3:
        kernel = np.ones((3, 3), np.uint8)
        mask_uint8 = cv2.morphologyEx(mask_uint8, cv2.MORPH_CLOSE, kernel, iterations=1)

    contours, _ = cv2.findContours(mask_uint8, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE)
    
    if not contours:
        return None
    
    contour = max(contours, key=cv2.contourArea)
    
    # 使用 approxPolyDP 简化多边形
    # 初始 epsilon 基于轮廓周长
    perimeter = cv2.arcLength(contour, True)
    epsilon = 0.001 * perimeter  # 初始精度
    
    # 逐步增加 epsilon 直到顶点数 <= max_points
    approx = cv2.approxPolyDP(contour, epsilon, True)
    while len(approx) > max_points and epsilon < 0.1 * perimeter:
        epsilon *= 1.5
        approx = cv2.approxPolyDP(contour, epsilon, True)
    
    points = []
    for pt in approx.squeeze():
        if len(pt.shape) == 0:  # 单点情况
            continue
        if pt.shape[0] == 2:
            points.append({"x": float(pt[0]), "y": float(pt[1])})
    
    if len(points) < 3:
        return None
    
    # 计算边界框
    x_coords = [p["x"] for p in points]
    y_coords = [p["y"] for p in points]
    x_min, x_max = min(x_coords), max(x_coords)
    y_min, y_max = min(y_coords), max(y_coords)
    
    if output_type == "rect":
        # 返回矩形框（归一化坐标）
        return {
            "type": "rect",
            "categoryName": "object",
            "confidence": float(score),
            "data": {
                "x": x_min / img_w,
                "y": y_min / img_h,
                "width": (x_max - x_min) / img_w,
                "height": (y_max - y_min) / img_h
            }
        }
    else:
        # 返回多边形（归一化坐标）
        points_normalized = [[p["x"] / img_w, p["y"] / img_h] for p in points]
        return {
            "type": "polygon",
            "categoryName": "object",
            "confidence": float(score),
            "data": {
                "points": points_normalized
            }
        }


def _handle_infer(request_id: str, payload: dict) -> None:
    """执行推理（带提示）"""
    if not _predictor:
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
        img_h, img_w = _current_image.shape[:2]
        output_type = payload.get("outputType", "polygon")  # "polygon" 或 "rect"
        
        # 推理模式
        point_coords = None
        point_labels = None
        box = None
        multimask_output = payload.get("multimask", True)
        
        if "points" in payload and payload["points"]:
            points = payload["points"]
            point_coords = np.array([[p["x"] * img_w, p["y"] * img_h] for p in points], dtype=np.float32)
            point_labels = np.array([1 if p.get("type") == "positive" else 0 for p in points], dtype=np.int32)
            _log(f"Converted points: {points} -> {point_coords.tolist()}")
        
        if "box" in payload and payload["box"]:
            b = payload["box"]
            box = np.array([
                b["x"] * img_w, 
                b["y"] * img_h, 
                (b["x"] + b["width"]) * img_w, 
                (b["y"] + b["height"]) * img_h
            ], dtype=np.float32)
        
        if point_coords is None and box is None:
            _log("No prompts provided, returning empty result")
            _send({
                "requestId": request_id,
                "cmd": "infer",
                "success": True,
                "annotations": []
            })
            return
        
        _log(f"Running inference with prompts: points={point_coords is not None}, box={box is not None}")
        
        # 执行预测
        masks, scores, logits = _predictor.predict(
            point_coords=point_coords,
            point_labels=point_labels,
            box=box,
            multimask_output=multimask_output
        )
        
        # 转换结果
        annotations = []
        
        # 如果是多掩码输出，选择得分最高的
        if multimask_output and len(masks) > 1:
            best_idx = np.argmax(scores)
            masks = [masks[best_idx]]
            scores = [scores[best_idx]]
        
        # 转换 mask 为标注
        for i, (mask, score) in enumerate(zip(masks, scores)):
            ann = _mask_to_annotation(mask, score, img_w, img_h, output_type)
            if ann:
                annotations.append(ann)
        
        _log(f"Inference complete: {len(annotations)} masks")
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
    global _predictor, _current_model_path, _current_image, _current_image_path
    
    _predictor = None
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
    _log("SAM-2 inference service started (stdio mode)")
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
