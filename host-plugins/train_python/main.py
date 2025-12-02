#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
EasyMark YOLOv8/YOLOv11 训练插件
基于 Ultralytics 框架实现目标检测、姿态估计、实例分割训练
"""
import argparse
import json
import socket
import sys
import threading
from pathlib import Path
from typing import Dict, Any, Optional


class SocketClient:
    """Socket 通信客户端"""
    
    def __init__(self, port: int):
        self.port = port
        self.sock: Optional[socket.socket] = None
        self.connected = False
        self._lock = threading.Lock()
        
    def connect(self) -> bool:
        """连接到宿主"""
        try:
            self.sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            self.sock.connect(('localhost', self.port))
            self.connected = True
            return True
        except Exception as e:
            print(f"[ERROR] Failed to connect to host: {e}", file=sys.stderr)
            return False
    
    def send(self, msg_type: str, payload: Dict[str, Any]):
        """发送消息"""
        if not self.connected or not self.sock:
            return
        with self._lock:
            try:
                msg = json.dumps({'type': msg_type, 'payload': payload}, ensure_ascii=False) + '\n'
                self.sock.sendall(msg.encode('utf-8'))
            except Exception as e:
                print(f"[ERROR] Failed to send message: {e}", file=sys.stderr)
    
    def recv(self) -> Optional[Dict]:
        """接收消息"""
        if not self.connected or not self.sock:
            return None
        try:
            data = b''
            while True:
                chunk = self.sock.recv(4096)
                if not chunk:
                    break
                data += chunk
                if b'\n' in data:
                    break
            if data:
                return json.loads(data.decode('utf-8').strip())
        except Exception as e:
            print(f"[ERROR] Failed to receive message: {e}", file=sys.stderr)
        return None
    
    def close(self):
        """关闭连接"""
        if self.sock:
            try:
                self.sock.close()
            except:
                pass
        self.connected = False


class TrainingPlugin:
    """训练插件主类"""
    
    def __init__(self, port: int, task_id: str):
        self.port = port
        self.task_id = task_id
        self.client = SocketClient(port)
        self.running = False
        self.trainer = None
        
    def log(self, level: str, text: str):
        """发送日志消息"""
        self.client.send('LOG', {'level': level, 'text': text})
        print(f"[{level.upper()}] {text}")
        
    def progress(self, epoch: int, total_epochs: int, batch: int,
                 total_batches: int, global_batch: int,
                 global_total_batches: int, metrics: Dict[str, float]):
        """发送进度消息"""
        if global_total_batches > 0:
            progress_val = global_batch / global_total_batches
        else:
            progress_val = 0.0

        self.client.send('PROGRESS', {
            'epoch': epoch,
            'totalEpochs': total_epochs,
            'batch': batch,
            'totalBatches': total_batches,
            'globalBatch': global_batch,
            'globalTotalBatches': global_total_batches,
            'progress': progress_val,
            'metrics': metrics
        })
        
    def epoch_end(self, epoch: int, metrics: Dict[str, float]):
        """发送 epoch 结束消息"""
        self.client.send('EPOCH_END', {
            'epoch': epoch,
            'metrics': metrics
        })
        
    def done(self, message: str, best_model: str, last_model: str, 
             metrics: Optional[Dict[str, float]] = None):
        """发送训练完成消息"""
        payload = {
            'message': message,
            'bestModel': best_model,
            'lastModel': last_model
        }
        if metrics:
            payload['metrics'] = metrics
        self.client.send('DONE', payload)
        
    def error(self, message: str, details: str = ''):
        """发送错误消息"""
        self.client.send('ERROR', {
            'message': message,
            'details': details
        })
        
    def train(self, config: Dict[str, Any]):
        """执行训练"""
        import os
        import io
        
        # 禁止联网下载（ultralytics 配置）
        os.environ['YOLO_OFFLINE'] = '1'
        os.environ['HUB_OFFLINE'] = '1'
        
        # 创建输出重定向类，将 stdout/stderr 转发给宿主
        class OutputRedirector(io.TextIOBase):
            def __init__(self, plugin, original, level='info'):
                self.plugin = plugin
                self.original = original
                self.level = level
                self.buffer = ''
                
            def write(self, text):
                if text:
                    # 写入原始输出
                    self.original.write(text)
                    self.original.flush()
                    # 缓冲并按行发送
                    self.buffer += text
                    while '\n' in self.buffer:
                        line, self.buffer = self.buffer.split('\n', 1)
                        line = line.strip()
                        if line:
                            self.plugin.client.send('OUTPUT', {'text': line})
                return len(text) if text else 0
                
            def flush(self):
                self.original.flush()
                if self.buffer.strip():
                    self.plugin.client.send('OUTPUT', {'text': self.buffer.strip()})
                    self.buffer = ''
        
        # 安装输出重定向
        old_stdout = sys.stdout
        old_stderr = sys.stderr
        sys.stdout = OutputRedirector(self, old_stdout, 'info')
        sys.stderr = OutputRedirector(self, old_stderr, 'error')
        
        try:
            from ultralytics import YOLO
        except ImportError as e:
            self.error('依赖缺失', f'无法导入 ultralytics: {e}')
            return
        finally:
            # 如果导入失败，恢复输出
            pass
            
        dataset_path = config['datasetPath']
        output_path = config['outputPath']
        model_path = config['modelPath']
        params = config['params']
        dataset_info = config.get('datasetInfo', {})
        
        # 创建输出目录
        Path(output_path).mkdir(parents=True, exist_ok=True)
        
        self.log('info', f'任务ID: {self.task_id}')
        self.log('info', f'数据集路径: {dataset_path}')
        self.log('info', f'模型路径: {model_path}')
        self.log('info', f'输出路径: {output_path}')
        self.log('info', f'训练参数: {json.dumps(params, ensure_ascii=False)}')
        
        # 加载模型
        self.log('info', f'正在加载模型...')
        try:
            model = YOLO(model_path)
        except Exception as e:
            self.error('模型加载失败', str(e))
            return
            
        self.log('info', '模型加载完成')
        
        # 准备训练参数
        epochs = params.get('epochs', 100)
        batch = params.get('batch', 16)
        imgsz = params.get('imgsz', 640)
        lr0 = params.get('lr0', 0.01)
        optimizer = params.get('optimizer', 'auto')
        augment = params.get('augment', True)
        patience = params.get('patience', 50)
        
        # 自动检测设备：优先使用 GPU，不可用时回退到 CPU
        import torch
        device = params.get('device', 'auto')
        if device == 'auto' or device == '0':
            if torch.cuda.is_available():
                device = '0'
                self.log('info', f'使用 GPU 设备: {torch.cuda.get_device_name(0)}')
            else:
                device = 'cpu'
                self.log('warning', 'CUDA 不可用，使用 CPU 训练（速度较慢）')
        else:
            self.log('info', f'使用指定设备: {device}')
        
        # 数据集配置文件路径
        data_yaml = Path(dataset_path) / 'data.yaml'
        if not data_yaml.exists():
            self.error('数据集配置缺失', f'未找到 data.yaml: {data_yaml}')
            return
            
        self.log('info', f'数据集配置: {data_yaml}')
        
        # 设置回调
        self.running = True
        batches_per_epoch = 0
        total_batches_global = 0  # 全局总 batch 数
        global_batch_counter = 0  # 全局 batch 计数器
        epoch_batch_counter = 0   # 当前 epoch 内的 batch 计数器
        last_epoch_index = None   # 上一次的 epoch 索引（0-based）

        def on_train_start(trainer):
            nonlocal batches_per_epoch, total_batches_global, global_batch_counter, epoch_batch_counter, last_epoch_index
            batches_per_epoch = len(trainer.train_loader) if trainer.train_loader else 100
            total_batches_global = epochs * batches_per_epoch
            global_batch_counter = 0
            epoch_batch_counter = 0
            last_epoch_index = None
            self.log('info', f'训练开始: {epochs} epochs, 每 epoch {batches_per_epoch} batches, 总计 {total_batches_global} batches')

        def on_train_batch_end(trainer):
            nonlocal batches_per_epoch, total_batches_global, global_batch_counter, epoch_batch_counter, last_epoch_index
            if not self.running:
                trainer.stop = True
                return

            # 确保 batches_per_epoch 已初始化（防止回调顺序问题）
            if batches_per_epoch == 0:
                batches_per_epoch = len(trainer.train_loader) if trainer.train_loader else 100
                total_batches_global = epochs * batches_per_epoch

            # 当前 epoch 索引（0-based）及 1-based 显示值
            epoch_index = getattr(trainer, 'epoch', 0)
            epoch_1_based = epoch_index + 1

            # 发生 epoch 切换时，重置当前 epoch 的 batch 计数器
            if last_epoch_index is None or epoch_index != last_epoch_index:
                last_epoch_index = epoch_index
                epoch_batch_counter = 0

            # 递增计数器
            epoch_batch_counter += 1
            global_batch_counter += 1
            current_batch = epoch_batch_counter

            # 构建指标（使用驼峰命名，与前端一致）
            metrics = {}
            if hasattr(trainer, 'loss_items') and trainer.loss_items is not None:
                loss_names = ['boxLoss', 'clsLoss', 'dflLoss']
                for i, name in enumerate(loss_names):
                    if i < len(trainer.loss_items):
                        metrics[name] = float(trainer.loss_items[i])
                # 计算总训练损失
                if len(trainer.loss_items) >= 3:
                    metrics['trainLoss'] = float(sum(trainer.loss_items[:3]))

            current_epoch = epoch_1_based

            self.progress(
                epoch=current_epoch,
                total_epochs=epochs,
                batch=current_batch,
                total_batches=batches_per_epoch,
                global_batch=global_batch_counter,
                global_total_batches=total_batches_global,
                metrics=metrics
            )
            
        def on_fit_epoch_end(trainer):
            """每个 epoch 结束后（包含验证后）触发"""
            if not self.running:
                return
                
            metrics = {}
            current_epoch = getattr(trainer, 'epoch', 0) + 1
            
            # 尝试多种方式获取验证指标
            # 方式1: trainer.metrics (验证后的结果)
            if hasattr(trainer, 'metrics') and trainer.metrics:
                m = trainer.metrics
                # DetMetrics 对象
                if hasattr(m, 'box'):
                    box = m.box
                    if hasattr(box, 'map50'):
                        metrics['mAP50'] = float(box.map50)
                    if hasattr(box, 'map'):
                        metrics['mAP5095'] = float(box.map)
                # 也可能是字典
                elif isinstance(m, dict):
                    if 'metrics/mAP50(B)' in m:
                        metrics['mAP50'] = float(m['metrics/mAP50(B)'])
                    if 'metrics/mAP50-95(B)' in m:
                        metrics['mAP5095'] = float(m['metrics/mAP50-95(B)'])
            
            # 方式2: trainer.validator.metrics
            if not metrics and hasattr(trainer, 'validator') and trainer.validator:
                v = trainer.validator
                if hasattr(v, 'metrics') and v.metrics:
                    vm = v.metrics
                    if hasattr(vm, 'box'):
                        box = vm.box
                        if hasattr(box, 'map50'):
                            metrics['mAP50'] = float(box.map50)
                        if hasattr(box, 'map'):
                            metrics['mAP5095'] = float(box.map)
            
            # 获取验证损失
            if hasattr(trainer, 'loss') and trainer.loss is not None:
                try:
                    if hasattr(trainer.loss, 'item'):
                        metrics['valLoss'] = float(trainer.loss.item())
                    elif hasattr(trainer.loss, '__iter__'):
                        metrics['valLoss'] = float(sum(trainer.loss))
                    else:
                        metrics['valLoss'] = float(trainer.loss)
                except:
                    pass
                    
            self.epoch_end(current_epoch, metrics)
            if metrics:
                self.log('info', f'Epoch {current_epoch}/{epochs} 验证: mAP50={metrics.get("mAP50", "-"):.4f}, mAP50-95={metrics.get("mAP5095", "-"):.4f}')
            
        def on_train_end(trainer):
            self.log('info', '训练流程结束')
            
        # 注册回调
        model.add_callback('on_train_start', on_train_start)
        model.add_callback('on_train_batch_end', on_train_batch_end)
        model.add_callback('on_fit_epoch_end', on_fit_epoch_end)
        model.add_callback('on_train_end', on_train_end)
        
        # 开始训练
        self.log('info', '开始训练...')
        
        try:
            # 构建训练参数
            train_kwargs = {
                'data': str(data_yaml),
                'epochs': epochs,
                'batch': batch,
                'imgsz': imgsz,
                'device': device,
                'lr0': lr0,
                'augment': augment,
                'patience': patience,
                'project': output_path,
                'name': 'train',
                'exist_ok': True,
                'verbose': True,
                'amp': False,  # 禁用 AMP 避免缓存问题
                'plots': True   # 启用 Ultralytics 绘图，在输出目录生成曲线图
            }
            # 只有明确指定优化器时才传入
            if optimizer and optimizer != 'auto':
                train_kwargs['optimizer'] = optimizer
            
            results = model.train(**train_kwargs)
            
            # 训练完成
            best_path = Path(output_path) / 'train' / 'weights' / 'best.pt'
            last_path = Path(output_path) / 'train' / 'weights' / 'last.pt'
            
            final_metrics = {}
            if results and hasattr(results, 'box'):
                final_metrics['best_mAP50'] = float(results.box.map50) if hasattr(results.box, 'map50') else 0
                final_metrics['best_mAP50-95'] = float(results.box.map) if hasattr(results.box, 'map') else 0
                
            self.done(
                message='训练完成',
                best_model='train/weights/best.pt' if best_path.exists() else '',
                last_model='train/weights/last.pt' if last_path.exists() else '',
                metrics=final_metrics
            )
            
        except KeyboardInterrupt:
            self.log('warning', '训练被用户中断')
            self.error('训练中断', '用户手动停止训练')
        except Exception as e:
            self.log('error', f'训练异常: {e}')
            self.error('训练失败', str(e))
        finally:
            self.running = False
            
    def handle_stop(self):
        """处理停止命令"""
        self.running = False
        self.log('info', '收到停止命令，正在停止训练...')
        
    def run(self):
        """主运行循环"""
        # 连接宿主
        if not self.client.connect():
            print("[ERROR] Cannot connect to host", file=sys.stderr)
            sys.exit(1)
            
        self.log('info', f'已连接到宿主 (端口 {self.port})')
        self.log('info', '等待训练指令...')
        
        try:
            while True:
                msg = self.client.recv()
                if not msg:
                    self.log('warning', '连接断开')
                    break
                    
                msg_type = msg.get('type', '')
                payload = msg.get('payload', {})
                
                if msg_type == 'START_TRAIN':
                    self.log('info', '收到 START_TRAIN 指令')
                    self.train(payload)
                    break
                elif msg_type == 'STOP_TRAIN':
                    self.handle_stop()
                    break
                else:
                    self.log('warning', f'未知消息类型: {msg_type}')
                    
        except Exception as e:
            self.log('error', f'运行异常: {e}')
            self.error('插件异常', str(e))
        finally:
            self.client.close()
            self.log('info', '插件退出')


def setup_ultralytics_font():
    """预置 Ultralytics 所需的字体，避免训练时下载"""
    import shutil
    import os
    
    # 插件目录下的字体
    plugin_dir = Path(__file__).parent
    font_src = plugin_dir / "Arial.Unicode.ttf"
    
    if not font_src.exists():
        return
    
    # Ultralytics 配置目录
    if sys.platform == "win32":
        config_dir = Path(os.environ.get("APPDATA", "")) / "Ultralytics"
    else:
        config_dir = Path.home() / ".config" / "Ultralytics"
    
    config_dir.mkdir(parents=True, exist_ok=True)
    font_dst = config_dir / "Arial.Unicode.ttf"
    
    if not font_dst.exists():
        try:
            shutil.copy2(font_src, font_dst)
            print(f"[INFO] Copied font to {font_dst}")
        except Exception as e:
            print(f"[WARN] Failed to copy font: {e}")


def main():
    # 预置字体
    setup_ultralytics_font()
    
    parser = argparse.ArgumentParser(description='EasyMark YOLOv8 Training Plugin')
    parser.add_argument('--socket-port', type=int, required=True, help='Host socket port')
    parser.add_argument('--task-id', type=str, required=True, help='Training task ID')
    args = parser.parse_args()
    
    print(f"[INFO] Starting training plugin...")
    print(f"[INFO] Socket port: {args.socket_port}")
    print(f"[INFO] Task ID: {args.task_id}")
    
    plugin = TrainingPlugin(args.socket_port, args.task_id)
    plugin.run()


if __name__ == '__main__':
    main()
