#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
EasyMark 推理服务（行协议版）- Ultralytics YOLO 插件
通过 stdin/stdout 与 Electron 主进程通信，每行一个 JSON
协议格式：
  请求: { "requestId": string, "cmd": "load_model"|"infer"|"unload"|"shutdown", ... }
  响应: { "requestId": string, "cmd": string, "success": bool, "error": string|null, ... }
"""
import json
import os
import sys
import time
from pathlib import Path

# 禁用 YOLO 的在线检查
os.environ.setdefault('YOLO_OFFLINE', '1')
os.environ.setdefault('ULTRALYTICS_SETUP', '0')
os.environ.setdefault('ULTRALYTICS_HUB', '0')
os.environ.setdefault('ULTRALYTICS_HUB_ENABLED', '0')

# 全局模型实例
_current_model = None
_current_model_path = None

# 数据根目录（由环境变量传入）
DATA_ROOT = os.environ.get("EASYMARK_DATA_PATH", "")


def _send(payload: dict) -> None:
    """以一行 JSON 的形式输出响应"""
    try:
        sys.stdout.write(json.dumps(payload, ensure_ascii=False) + "\n")
        sys.stdout.flush()
    except Exception:
        raise SystemExit(1)


def _log(msg: str) -> None:
    """输出日志到 stderr（不影响 stdout 的 JSON 协议）"""
    try:
        sys.stderr.write(f"[infer_service] {msg}\n")
        sys.stderr.flush()
    except Exception:
        pass


def _handle_load_model(request_id: str, weights: str) -> None:
    """加载模型"""
    global _current_model, _current_model_path
    
    if not weights:
        _send({
            'requestId': request_id,
            'cmd': 'load_model',
            'success': False,
            'error': 'weights path is required'
        })
        return
    
    weights_path = Path(weights).expanduser().resolve()
    if not weights_path.is_file():
        _send({
            'requestId': request_id,
            'cmd': 'load_model',
            'success': False,
            'error': f'Weights file not found: {weights_path}'
        })
        return
    
    # 如果已加载相同模型，跳过
    if _current_model is not None and _current_model_path == str(weights_path):
        _send({
            'requestId': request_id,
            'cmd': 'load_model',
            'success': True,
            'error': None
        })
        return
    
    try:
        from ultralytics import YOLO
        _log(f"Loading model: {weights_path}")
        _current_model = YOLO(str(weights_path))
        _current_model_path = str(weights_path)
        
        task = getattr(_current_model, 'task', 'detect')
        _log(f"Model loaded, task: {task}")
        
        _send({
            'requestId': request_id,
            'cmd': 'load_model',
            'success': True,
            'error': None,
            'task': task
        })
    except Exception as e:
        _current_model = None
        _current_model_path = None
        _send({
            'requestId': request_id,
            'cmd': 'load_model',
            'success': False,
            'error': f'Failed to load model: {e}'
        })


def _handle_unload(request_id: str) -> None:
    """卸载模型"""
    global _current_model, _current_model_path
    _current_model = None
    _current_model_path = None
    _send({
        'requestId': request_id,
        'cmd': 'unload',
        'success': True,
        'error': None
    })


def _parse_results(result) -> list:
    """解析 YOLO 推理结果（兼容 detect/pose/segment 三种模型）"""
    annotations = []
    
    try:
        # 获取图片尺寸（用于归一化）
        orig_shape = result.orig_shape  # (height, width)
        img_h, img_w = orig_shape[0], orig_shape[1]
        
        # 获取类别名称
        names = result.names if hasattr(result, 'names') else {}
        
        # 安全检查 masks
        has_masks = False
        try:
            if hasattr(result, 'masks') and result.masks is not None and len(result.masks) > 0:
                has_masks = True
        except Exception:
            has_masks = False
        
        # 检测框 (bbox) - 分割模型时不输出 bbox，只输出 polygon
        if not has_masks:
            boxes = getattr(result, 'boxes', None)
            if boxes is not None and len(boxes) > 0:
                for i in range(len(boxes)):
                    try:
                        # 获取边界框坐标 (xyxy 格式)
                        xyxy = boxes.xyxy[i].cpu().numpy()
                        x1, y1, x2, y2 = xyxy[0], xyxy[1], xyxy[2], xyxy[3]
                        
                        # 归一化坐标
                        nx = float(x1 / img_w)
                        ny = float(y1 / img_h)
                        nw = float((x2 - x1) / img_w)
                        nh = float((y2 - y1) / img_h)
                        
                        # 置信度
                        conf = float(boxes.conf[i].cpu().numpy())
                        
                        # 类别
                        cls_id = int(boxes.cls[i].cpu().numpy())
                        cls_name = names.get(cls_id, str(cls_id))
                        
                        ann = {
                            'type': 'bbox',
                            'categoryName': cls_name,
                            'confidence': conf,
                            'data': {
                                'x': nx,
                                'y': ny,
                                'width': nw,
                                'height': nh
                            }
                        }
                        
                        # 如果有关键点，添加到 bbox（pose 模型）
                        try:
                            if hasattr(result, 'keypoints') and result.keypoints is not None and i < len(result.keypoints):
                                kpts = result.keypoints[i]
                                if kpts is not None and hasattr(kpts, 'xy') and kpts.xy is not None:
                                    keypoints = []
                                    xy = kpts.xy.cpu().numpy()[0] if len(kpts.xy.shape) > 2 else kpts.xy.cpu().numpy()
                                    conf_arr = kpts.conf.cpu().numpy()[0] if hasattr(kpts, 'conf') and kpts.conf is not None else None
                                    
                                    for j in range(len(xy)):
                                        kx, ky = float(xy[j][0]), float(xy[j][1])
                                        # 归一化
                                        kx_norm = kx / img_w
                                        ky_norm = ky / img_h
                                        # 可见性
                                        if conf_arr is not None and j < len(conf_arr):
                                            v = 2 if float(conf_arr[j]) > 0.5 else 1
                                        else:
                                            v = 2 if (kx > 0 and ky > 0) else 0
                                        keypoints.append([kx_norm, ky_norm, v])
                                    
                                    if keypoints:
                                        ann['data']['keypoints'] = keypoints
                        except Exception as kp_err:
                            _log(f"Failed to parse keypoints: {kp_err}")
                        
                        annotations.append(ann)
                    except Exception as box_err:
                        _log(f"Failed to parse box {i}: {box_err}")
        
        # 分割掩码 (polygon)
        if has_masks:
            try:
                masks = result.masks
                for i in range(len(masks)):
                    if hasattr(masks, 'xyn') and masks.xyn is not None:
                        xyn = masks.xyn[i]
                        if len(xyn) > 0:
                            points = [[float(p[0]), float(p[1])] for p in xyn]
                            
                            # 类别和置信度
                            boxes = getattr(result, 'boxes', None)
                            cls_id = int(boxes.cls[i].cpu().numpy()) if boxes is not None else 0
                            cls_name = names.get(cls_id, str(cls_id))
                            conf = float(boxes.conf[i].cpu().numpy()) if boxes is not None else 1.0
                            
                            # 简化过多的点
                            if len(points) > 100:
                                step = len(points) // 50
                                points = points[::step]
                            
                            ann = {
                                'type': 'polygon',
                                'categoryName': cls_name,
                                'confidence': conf,
                                'data': {
                                    'points': points
                                }
                            }
                            annotations.append(ann)
            except Exception as mask_err:
                _log(f"Failed to parse masks: {mask_err}")
        
    except Exception as e:
        _log(f"Failed to parse results: {e}")
    
    return annotations


def _handle_infer(request_id: str, data: dict) -> None:
    """执行推理"""
    global _current_model
    
    if _current_model is None:
        _send({
            'requestId': request_id,
            'cmd': 'infer',
            'success': False,
            'error': 'model is not loaded',
            'annotations': []
        })
        return
    
    # 解析图片路径
    image_source = None
    
    # 优先使用 projectId + path 拼接
    project_id = data.get('projectId')
    rel_path = data.get('path')
    
    if DATA_ROOT and project_id and rel_path:
        rel_str = str(rel_path).replace('\\', '/').lstrip('/')
        if '..' in rel_str:
            _send({
                'requestId': request_id,
                'cmd': 'infer',
                'success': False,
                'error': 'path_invalid',
                'annotations': []
            })
            return
        base_dir = os.path.join(DATA_ROOT, 'project_item', str(project_id))
        image_source = os.path.join(base_dir, rel_str.replace('/', os.sep))
    elif data.get('imagePath'):
        image_source = data.get('imagePath')
    
    if not image_source:
        _send({
            'requestId': request_id,
            'cmd': 'infer',
            'success': False,
            'error': 'missing image path',
            'annotations': []
        })
        return
    
    if not os.path.exists(image_source):
        _send({
            'requestId': request_id,
            'cmd': 'infer',
            'success': False,
            'error': f'image not found: {image_source}',
            'annotations': []
        })
        return
    
    conf = data.get('conf', 0.25)
    iou = data.get('iou', 0.45)
    
    try:
        import traceback
        start_time = time.perf_counter()
        _log(f"Running inference on: {image_source}, conf={conf}, iou={iou}")
        
        try:
            results = _current_model.predict(
                source=image_source,
                conf=conf,
                iou=iou,
                verbose=False
            )
        except Exception as pred_err:
            _log(f"predict() failed: {pred_err}")
            _log(f"Traceback:\n{traceback.format_exc()}")
            raise
        
        infer_time_ms = (time.perf_counter() - start_time) * 1000.0
        
        if not results or len(results) == 0:
            _log(f"Inference finished in {infer_time_ms:.1f} ms, 0 annotations")
            _send({
                'requestId': request_id,
                'cmd': 'infer',
                'success': True,
                'error': None,
                'annotations': [],
                'inferTimeMs': infer_time_ms
            })
            return
        
        annotations = _parse_results(results[0])
        _log(f"Inference finished in {infer_time_ms:.1f} ms, {len(annotations)} annotations")
        
        _send({
            'requestId': request_id,
            'cmd': 'infer',
            'success': True,
            'error': None,
            'annotations': annotations,
            'inferTimeMs': infer_time_ms
        })
        
    except Exception as e:
        _log(f"Inference failed: {e}")
        _send({
            'requestId': request_id,
            'cmd': 'infer',
            'success': False,
            'error': f'inference failed: {e}',
            'annotations': []
        })


def main() -> None:
    """主循环：每行一个 JSON 命令"""
    plugin_id = os.environ.get("EASYMARK_PLUGIN_ID", "")
    plugin_path = os.environ.get("EASYMARK_PLUGIN_PATH", "")
    
    _log("========================================")
    _log("Ultralytics YOLO Inference Plugin Started")
    _log(f"Plugin ID: {plugin_id or '(not set)'}")
    _log(f"Plugin Path: {plugin_path or '(not set)'}")
    _log(f"Data Path: {DATA_ROOT or '(not set)'}")
    _log("========================================")
    
    while True:
        try:
            line = sys.stdin.readline()
        except Exception:
            break
        
        if not line:
            break
        
        line = line.strip()
        if not line:
            continue
        
        try:
            message = json.loads(line)
        except Exception:
            continue
        
        cmd = message.get('cmd')
        request_id = message.get('requestId', '')
        
        if cmd == 'load_model':
            _handle_load_model(request_id, message.get('weights'))
        elif cmd == 'infer':
            _handle_infer(request_id, message)
        elif cmd == 'unload':
            _handle_unload(request_id)
        elif cmd == 'shutdown':
            _send({
                'requestId': request_id,
                'cmd': 'shutdown',
                'success': True,
                'error': None
            })
            break
        else:
            _send({
                'requestId': request_id,
                'cmd': cmd,
                'success': False,
                'error': f'Unknown command: {cmd}'
            })
    
    _log("Inference service stopped")


if __name__ == '__main__':
    main()
