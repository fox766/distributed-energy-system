<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>分布式能源交易系统</title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
        }
        
        :root {
            --primary: #4361ee;
            --secondary: #3f37c9;
            --accent: #4895ef;
            --success: #4cc9f0;
            --energy-green: #2ecc71;
            --energy-yellow: #f39c12;
            --energy-blue: #3498db;
            --light: #f8f9fa;
            --dark: #212529;
            --gray: #6c757d;
            --danger: #e63946;
            --border-radius: 12px;
            --box-shadow: 0 10px 20px rgba(0, 0, 0, 0.1);
            --transition: all 0.3s ease;
        }
        
        body {
            background: linear-gradient(135deg, #f5f7fa 0%, #e4edf5 100%);
            min-height: 100vh;
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            color: var(--dark);
            padding-top: 60px;
        }
        
        .navbar {
            background: linear-gradient(to right, var(--primary), var(--accent));
            box-shadow: var(--box-shadow);
            position: fixed;
            top: 0;
            width: 100%;
            z-index: 1000;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        
        .card {
            border-radius: var(--border-radius);
            box-shadow: var(--box-shadow);
            border: none;
            margin-bottom: 20px;
            transition: var(--transition);
            background: white;
        }
        
        .card:hover {
            transform: translateY(-5px);
        }
        
        .card-header {
            background: linear-gradient(to right, var(--primary), var(--accent));
            color: white;
            border-radius: var(--border-radius) var(--border-radius) 0 0 !important;
            border-bottom: none;
            font-weight: 600;
            padding: 15px 20px;
        }
        
        .card-body {
            padding: 20px;
        }
        
        .section-title {
            color: var(--primary);
            border-left: 4px solid var(--accent);
            padding-left: 15px;
            margin: 25px 0 15px;
            font-size: 1.5rem;
        }
        
        .energy-grid {
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background-image: 
                linear-gradient(rgba(67, 97, 238, 0.03) 1px, transparent 1px),
                linear-gradient(90deg, rgba(67, 97, 238, 0.03) 1px, transparent 1px);
            background-size: 30px 30px;
            z-index: -1;
            opacity: 0.5;
        }
        
        .btn {
            padding: 10px 16px;
            border: none;
            border-radius: 8px;
            font-weight: 600;
            cursor: pointer;
            transition: var(--transition);
            display: inline-flex;
            align-items: center;
            gap: 8px;
        }
        
        .btn-primary {
            background: linear-gradient(to right, var(--primary), var(--accent));
            color: white;
        }
        
        .btn-success {
            background: linear-gradient(to right, var(--energy-green), #27ae60);
            color: white;
        }
        
        .btn-warning {
            background: linear-gradient(to right, var(--energy-yellow), #e67e22);
            color: white;
        }
        
        .btn-danger {
            background: linear-gradient(to right, var(--danger), #c0392b);
            color: white;
        }
        
        .btn-outline {
            background: transparent;
            border: 2px solid var(--gray);
            color: var(--gray);
        }
        
        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 10px rgba(0, 0, 0, 0.15);
        }
        
        .form-control {
            width: 100%;
            padding: 12px 15px;
            border: 2px solid #e1e5eb;
            border-radius: 8px;
            font-size: 16px;
            transition: var(--transition);
            outline: none;
        }
        
        .form-control:focus {
            border-color: var(--accent);
            box-shadow: 0 0 0 3px rgba(67, 97, 238, 0.15);
        }
        
        .status-badge {
            padding: 5px 12px;
            border-radius: 20px;
            font-weight: 500;
            font-size: 0.85rem;
            display: inline-block;
        }
        
        .status-created {
            background-color: rgba(52, 152, 219, 0.2);
            color: var(--energy-blue);
        }
        
        .status-matched {
            background-color: rgba(243, 156, 18, 0.2);
            color: var(--energy-yellow);
        }
        
        .status-finished {
            background-color: rgba(46, 204, 113, 0.2);
            color: var(--energy-green);
        }
        
        .user-card {
            background: linear-gradient(135deg, #ffffff 0%, #f7fff0 100%);
            border-left: 4px solid var(--energy-green);
        }
        
        .order-card {
            background: linear-gradient(135deg, #ffffff 0%, #fff5f0 100%);
            border-left: 4px solid var(--energy-yellow);
        }
        
        .dashboard-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        
        .stat-card {
            text-align: center;
            padding: 20px;
            border-radius: var(--border-radius);
            box-shadow: var(--box-shadow);
            color: white;
        }
        
        .stat-card i {
            font-size: 2.5rem;
            margin-bottom: 15px;
            opacity: 0.8;
        }
        
        .stat-card .number {
            font-size: 1.8rem;
            font-weight: 700;
            margin: 10px 0;
        }
        
        .stat-card .label {
            font-size: 1rem;
            opacity: 0.9;
        }
        
        .login-container {
            max-width: 400px;
            margin: 0 auto;
            padding: 20px;
        }
        
        .message {
            padding: 12px;
            border-radius: 8px;
            margin: 15px 0;
            font-size: 14px;
        }
        
        .error-message {
            background-color: rgba(230, 57, 70, 0.1);
            color: var(--danger);
        }
        
        .success-message {
            background-color: rgba(76, 201, 240, 0.1);
            color: var(--success);
        }
        
        .page-section {
            display: none;
        }
        
        .page-section.active {
            display: block;
            animation: fadeIn 0.5s ease;
        }
        
        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(20px); }
            to { opacity: 1; transform: translateY(0); }
        }
        
        .order-table {
            width: 100%;
            border-collapse: collapse;
        }
        
        .order-table th, .order-table td {
            padding: 12px 15px;
            text-align: left;
            border-bottom: 1px solid #e1e5eb;
        }
        
        .order-table th {
            background-color: #f8f9fa;
            font-weight: 600;
        }
        
        .order-table tr:hover {
            background-color: #f8f9fa;
        }
        
        .action-cell {
            display: flex;
            gap: 8px;
        }
        
        .user-info-panel {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 20px;
        }
        
        @media (max-width: 768px) {
            .user-info-panel {
                grid-template-columns: 1fr;
            }
            
            .dashboard-grid {
                grid-template-columns: 1fr;
            }
        }
        
        .energy-icon {
            background: rgba(67, 97, 238, 0.1);
            width: 50px;
            height: 50px;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            margin-right: 15px;
            color: var(--primary);
            font-size: 1.5rem;
        }
    </style>
</head>
<body>
    <div class="energy-grid"></div>
    
    <!-- 顶部导航栏 -->
    <nav class="navbar navbar-expand-lg navbar-dark bg-primary shadow-sm">
    <div class="container">
        <a class="navbar-brand" href="#">
            <i class="fas fa-bolt me-2"></i>
            <strong>分布式能源交易系统</strong>
        </a>
        <button class="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbarNav" aria-controls="navbarNav" aria-expanded="false" aria-label="Toggle navigation">
            <span class="navbar-toggler-icon"></span>
        </button>
        <div class="collapse navbar-collapse" id="navbarNav">
            <ul class="navbar-nav ms-auto">
                <li class="nav-item">
                    <a class="nav-link active" href="#" data-page="dashboard">
                        <i class="fas fa-tachometer-alt me-1"></i> 仪表盘
                    </a>
                </li>
                <li class="nav-item">
                    <a class="nav-link" href="#" data-page="orders">
                        <i class="fas fa-exchange-alt me-1"></i> 订单管理
                    </a>
                </li>
                <li class="nav-item">
                    <a class="nav-link" href="#" data-page="create-order">
                        <i class="fas fa-plus-circle me-1"></i> 创建订单
                    </a>
                </li>
                <li class="nav-item">
                    <a class="nav-link" href="#" data-page="user-info">
                        <i class="fas fa-user me-1"></i> 用户信息
                    </a>
                </li>
            </ul>
            <div class="d-flex">
                <div class="dropdown">
                    <button class="btn btn-outline-light dropdown-toggle" type="button" id="userDropdown" data-bs-toggle="dropdown" aria-expanded="false">
                        <i class="fas fa-user-circle me-1"></i> <span id="current-user">未登录</span>
                    </button>
                    <ul class="dropdown-menu dropdown-menu-end" aria-labelledby="userDropdown">
                        <li><a class="dropdown-item" href="#" id="logoutBtn"><i class="fas fa-sign-out-alt me-2"></i> 退出登录</a></li>
                    </ul>
                </div>
            </div>
        </div>
    </div>
</nav>


    <!-- 主内容区 -->
    <div class="container">
        <!-- 登录界面 -->
        <div class="page-section active" id="login-section">
            <div class="login-container">
                <div class="card">
                    <div class="card-body text-center">
                        <div class="mb-4">
                            <div class="energy-icon mx-auto">
                                <i class="fas fa-bolt"></i>
                            </div>
                            <h3 class="mt-3">分布式能源交易系统</h3>
                            <p class="text-muted">请登录您的账户</p>
                        </div>
                        
                        <div id="error-message" class="message error-message" style="display: none;"></div>
                        <div id="success-message" class="message success-message" style="display: none;"></div>
                        
                        <div class="mb-3">
                            <label for="username" class="form-label">用户名</label>
                            <div class="input-group">
                                <span class="input-group-text"><i class="fas fa-user"></i></span>
                                <input type="text" class="form-control" id="username" placeholder="请输入用户名">
                            </div>
                        </div>
                        
                        <div class="mb-3">
                            <label for="password" class="form-label">密码</label>
                            <div class="input-group">
                                <span class="input-group-text"><i class="fas fa-lock"></i></span>
                                <input type="password" class="form-control" id="password" placeholder="请输入密码">
                                <button class="btn btn-outline-secondary" type="button" id="togglePassword">
                                    <i class="fas fa-eye"></i>
                                </button>
                            </div>
                        </div>
                        
                        <div class="d-grid gap-2 mt-4">
                            <button class="btn btn-primary btn-lg" id="loginBtn">
                                <i class="fas fa-sign-in-alt me-2"></i>登录
                            </button>
                            <button class="btn btn-outline-primary btn-lg" id="showRegisterBtn">
                                <i class="fas fa-user-plus me-2"></i>注册
                            </button>
                        </div>
                        
                        <div class="mt-3 text-center">
                            <p class="text-muted">没有账户? <a href="#" class="text-primary" id="showRegisterLink">立即注册</a></p>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        
        <!-- 仪表盘界面 -->
        <div class="page-section" id="dashboard-section">
            <h2 class="section-title">能源交易仪表盘</h2>
            
            <div class="dashboard-grid">
                <div class="stat-card" style="background: linear-gradient(135deg, var(--primary), var(--secondary));">
                    <i class="fas fa-bolt"></i>
                    <div class="number" id="available-energy">0.0 kWh</div>
                    <div class="label">可用能源</div>
                </div>
                <div class="stat-card" style="background: linear-gradient(135deg, var(--energy-green), #27ae60);">
                    <i class="fas fa-wallet"></i>
                    <div class="number" id="user-balance">¥0.00</div>
                    <div class="label">账户余额</div>
                </div>
                <div class="stat-card" style="background: linear-gradient(135deg, var(--energy-yellow), #e67e22);">
                    <i class="fas fa-exchange-alt"></i>
                    <div class="number" id="active-orders">0</div>
                    <div class="label">活跃订单</div>
                </div>
                <div class="stat-card" style="background: linear-gradient(135deg, var(--energy-blue), #2980b9);">
                    <i class="fas fa-chart-line"></i>
                    <div class="number" id="energy-price">¥0.00</div>
                    <div class="label">当前能源价格</div>
                </div>
            </div>
            
            <h3 class="section-title">最新订单</h3>
            <div class="card">
                <div class="card-body">
                    <div class="table-responsive">
                        <table class="order-table">
                            <thead>
                                <tr>
                                    <th>订单ID</th>
                                    <th>类型</th>
                                    <th>数量 (kWh)</th>
                                    <th>价格</th>
                                    <th>状态</th>
                                    <th>创建时间</th>
                                    <th>操作</th>
                                </tr>
                            </thead>
                            <tbody id="orders-table">
                                <tr>
                                    <td colspan="7" class="text-center py-5">加载中...</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>
        
        <!-- 订单管理界面 -->
        <div class="page-section" id="orders-section">
            <h2 class="section-title">订单管理</h2>
            
            <div class="card">
                <div class="card-header">
                    <i class="fas fa-filter me-2"></i> 筛选条件
                </div>
                <div class="card-body">
                    <div class="row">
                        <div class="col-md-4 mb-3">
                            <label class="form-label">订单状态</label>
                            <select class="form-control" id="order-status-filter">
                                <option value="">全部状态</option>
                                <option value="CREATED">已创建</option>
                                <option value="MATCHED">已匹配</option>
                                <option value="FINISHED">已完成</option>
                            </select>
                        </div>
                        <div class="col-md-4 mb-3">
                            <label class="form-label">订单类型</label>
                            <select class="form-control" id="order-type-filter">
                                <option value="">全部类型</option>
                                <option value="sell">出售订单</option>
                                <option value="buy">购买订单</option>
                            </select>
                        </div>
                        <div class="col-md-4 mb-3 d-flex align-items-end">
                            <button class="btn btn-primary w-100" id="apply-filter-btn">
                                <i class="fas fa-filter me-2"></i>应用筛选
                            </button>
                        </div>
                    </div>
                </div>
            </div>
            
            <div class="card">
                <div class="card-header d-flex justify-content-between align-items-center">
                    <div>
                        <i class="fas fa-list me-2"></i> 订单列表
                    </div>
                    <button class="btn btn-sm btn-outline-primary" id="refresh-orders-btn">
                        <i class="fas fa-sync-alt"></i> 刷新
                    </button>
                </div>
                <div class="card-body">
                    <div class="table-responsive">
                        <table class="order-table">
                            <thead>
                                <tr>
                                    <th>订单ID</th>
                                    <th>卖方</th>
                                    <th>买方</th>
                                    <th>数量 (kWh)</th>
                                    <th>价格</th>
                                    <th>状态</th>
                                    <th>创建时间</th>
                                    <th>操作</th>
                                </tr>
                            </thead>
                            <tbody id="all-orders-table">
                                <tr>
                                    <td colspan="8" class="text-center py-5">加载中...</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>
        
        <!-- 创建订单界面 -->
        <div class="page-section" id="create-order-section">
            <h2 class="section-title">创建新订单</h2>
            
            <div class="card">
                <div class="card-header">
                    <i class="fas fa-plus-circle me-2"></i> 订单信息
                </div>
                <div class="card-body">
                    <div class="row">
                        <div class="col-md-6">
                            <div class="mb-3">
                                <label class="form-label">交易类型</label>
                                <div class="d-flex gap-2">
                                    <button class="btn btn-outline-primary flex-grow-1 active" type="button" data-type="sell">
                                        <i class="fas fa-arrow-up me-2"></i>出售能源
                                    </button>
                                    <button class="btn btn-outline-success flex-grow-1" type="button" data-type="buy">
                                        <i class="fas fa-arrow-down me-2"></i>购买能源
                                    </button>
                                </div>
                            </div>
                            
                            <div class="mb-3">
                                <label for="order-amount" class="form-label">能源数量 (kWh)</label>
                                <input type="range" class="form-range" min="1" max="1000" id="order-amount-range" value="100">
                                <div class="input-group mt-2">
                                    <input type="number" class="form-control" id="order-amount" value="100" min="1" max="1000">
                                    <span class="input-group-text">kWh</span>
                                </div>
                            </div>
                        </div>
                        
                        <div class="col-md-6">
                            <div class="mb-4">
                                <label class="form-label">订单详情</label>
                                <div class="card order-card p-3">
                                    <div class="d-flex justify-content-between mb-2">
                                        <span>能源单价:</span>
                                        <span id="order-price">¥1.00</span>
                                    </div>
                                    <div class="d-flex justify-content-between mb-2">
                                        <span>交易费用:</span>
                                        <span id="order-fee">¥0.01</span>
                                    </div>
                                    <div class="d-flex justify-content-between mb-2">
                                        <span>总金额:</span>
                                        <span id="order-total">¥100.00</span>
                                    </div>
                                </div>
                            </div>
                            
                            <button class="btn btn-primary w-100" id="create-order-btn">
                                <i class="fas fa-check-circle me-2"></i>创建订单
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        
        <!-- 用户信息界面 -->
        <div class="page-section" id="user-info-section">
            <h2 class="section-title">用户信息</h2>
            
            <div class="card">
                <div class="card-header">
                    <i class="fas fa-user me-2"></i> 基本信息
                </div>
                <div class="card-body user-card">
                    <div class="user-info-panel">
                        <div>
                            <div class="d-flex align-items-center mb-4">
                                <div class="energy-icon">
                                    <i class="fas fa-user"></i>
                                </div>
                                <div>
                                    <h4 id="user-name">未登录</h4>
                                    <p class="text-muted mb-0" id="user-id">ID: -</p>
                                </div>
                            </div>
                            
                            <div class="mb-3">
                                <h5><i class="fas fa-id-card me-2"></i> 账户信息</h5>
                                <div class="mt-3">
                                    <div class="d-flex justify-content-between mb-2">
                                        <span>账户余额:</span>
                                        <span id="user-balance-info">¥0.00</span>
                                    </div>
                                    <div class="d-flex justify-content-between mb-2">
                                        <span>可用能源:</span>
                                        <span id="user-available">0.0 kWh</span>
                                    </div>
                                    <div class="d-flex justify-content-between mb-2">
                                        <span>用户角色:</span>
                                        <span id="user-role">普通用户</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                        
                        <div>
                            <h5><i class="fas fa-chart-pie me-2"></i> 交易统计</h5>
                            <div class="mt-4">
                                <div class="d-flex justify-content-between mb-2">
                                    <span>总订单数:</span>
                                    <span id="total-orders">0</span>
                                </div>
                                <div class="d-flex justify-content-between mb-2">
                                    <span>已完成订单:</span>
                                    <span id="completed-orders">0</span>
                                </div>
                                <div class="d-flex justify-content-between mb-2">
                                    <span>进行中订单:</span>
                                    <span id="active-orders-count">0</span>
                                </div>
                                <div class="d-flex justify-content-between mb-2">
                                    <span>总交易量:</span>
                                    <span id="total-energy">0.0 kWh</span>
                                </div>
                            </div>
                            
                            <div class="mt-4">
                                <button class="btn btn-outline-primary w-100" id="update-balance-btn">
                                    <i class="fas fa-sync-alt me-2"></i> 更新账户信息
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            
            <h3 class="section-title mt-4">我的订单</h3>
            <div class="card">
                <div class="card-body">
                    <div class="table-responsive">
                        <table class="order-table">
                            <thead>
                                <tr>
                                    <th>订单ID</th>
                                    <th>数量 (kWh)</th>
                                    <th>价格</th>
                                    <th>状态</th>
                                    <th>创建时间</th>
                                    <th>操作</th>
                                </tr>
                            </thead>
                            <tbody id="user-orders-table">
                                <tr>
                                    <td colspan="6" class="text-center py-5">加载中...</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- 订单详情模态框 -->
    <div class="modal" id="order-detail-modal" tabindex="-1">
        <div class="modal-dialog modal-lg">
            <div class="modal-content">
                <div class="modal-header">
                    <h5 class="modal-title">订单详情</h5>
                    <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
                </div>
                <div class="modal-body">
                    <div class="card mb-3">
                        <div class="card-body">
                            <div class="row">
                                <div class="col-md-6">
                                    <div class="mb-3">
                                        <label class="form-label">订单ID</label>
                                        <div id="modal-order-id">-</div>
                                    </div>
                                    <div class="mb-3">
                                        <label class="form-label">卖方</label>
                                        <div id="modal-party-a">-</div>
                                    </div>
                                    <div class="mb-3">
                                        <label class="form-label">创建时间</label>
                                        <div id="modal-created-at">-</div>
                                    </div>
                                </div>
                                <div class="col-md-6">
                                    <div class="mb-3">
                                        <label class="form-label">订单状态</label>
                                        <div>
                                            <span class="status-badge" id="modal-order-status">-</span>
                                        </div>
                                    </div>
                                    <div class="mb-3">
                                        <label class="form-label">买方</label>
                                        <div id="modal-party-b">-</div>
                                    </div>
                                    <div class="mb-3">
                                        <label class="form-label">交易金额</label>
                                        <div id="modal-total-amount">¥0.00</div>
                                    </div>
                                </div>
                            </div>
                            <div class="row mt-3">
                                <div class="col-md-4">
                                    <div class="card p-2 text-center">
                                        <div class="text-muted">能源数量</div>
                                        <div class="fs-5 fw-bold" id="modal-energy-amount">0.0 kWh</div>
                                    </div>
                                </div>
                                <div class="col-md-4">
                                    <div class="card p-2 text-center">
                                        <div class="text-muted">能源单价</div>
                                        <div class="fs-5 fw-bold" id="modal-energy-price">¥0.00</div>
                                    </div>
                                </div>
                                <div class="col-md-4">
                                    <div class="card p-2 text-center">
                                        <div class="text-muted">交易费用</div>
                                        <div class="fs-5 fw-bold" id="modal-transaction-fee">¥0.00</div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                    
                    <div class="d-flex justify-content-between" id="order-actions">
                        <button class="btn btn-success" id="match-order-btn">
                            <i class="fas fa-handshake me-2"></i>匹配订单
                        </button>
                        <button class="btn btn-primary" id="settle-order-btn">
                            <i class="fas fa-check-circle me-2"></i>结算订单
                        </button>
                        <button class="btn btn-danger" id="cancel-order-btn">
                            <i class="fas fa-times-circle me-2"></i>取消订单
                        </button>
                    </div>
                </div>
                <div class="modal-footer">
                    <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">关闭</button>
                </div>
            </div>
        </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/js/bootstrap.bundle.min.js"></script>
    <script>
        // 后端API基础URL
        const API_BASE_URL = 'http://localhost:8080';
        
        // 当前用户信息
        let currentUser = null;
        let currentToken = null;
        
        // 页面初始化
        document.addEventListener('DOMContentLoaded', function() {
            // 初始化页面
            initPage();
            
            // 绑定事件
            bindEvents();
        });
        
        // 初始化页面
        function initPage() {
            // 隐藏所有页面，只显示登录页
            document.querySelectorAll('.page-section').forEach(section => {
                section.classList.remove('active');
            });
            document.getElementById('login-section').classList.add('active');
            
            // 初始化订单金额计算
            updateOrderDetails();
        }
        
        // 绑定事件
        function bindEvents() {
            // 导航菜单点击
            document.querySelectorAll('#main-nav .nav-link').forEach(link => {
                link.addEventListener('click', function(e) {
                    e.preventDefault();
                    const page = this.getAttribute('data-page');
                    showPage(page);
                    
                    // 更新活动状态
                    document.querySelectorAll('#main-nav .nav-link').forEach(el => {
                        el.classList.remove('active');
                    });
                    this.classList.add('active');
                });
            });
            
            // 密码可见性切换
            document.getElementById('togglePassword').addEventListener('click', function() {
                const passwordInput = document.getElementById('password');
                const type = passwordInput.getAttribute('type') === 'password' ? 'text' : 'password';
                passwordInput.setAttribute('type', type);
                this.innerHTML = type === 'password' ? '<i class="fas fa-eye"></i>' : '<i class="fas fa-eye-slash"></i>';
            });
            
            // 登录按钮
            document.getElementById('loginBtn').addEventListener('click', loginUser);
            
            // 注册按钮
            document.getElementById('showRegisterBtn').addEventListener('click', function() {
                const username = document.getElementById('username').value;
                const password = document.getElementById('password').value;
                
                if (!username || !password) {
                    showError('请输入用户名和密码');
                    return;
                }
                
                registerUser(username, password);
            });
            
            // 显示注册链接
            document.getElementById('showRegisterLink').addEventListener('click', function(e) {
                e.preventDefault();
                document.getElementById('showRegisterBtn').click();
            });
            
            // 能源数量滑块
            const amountRange = document.getElementById('order-amount-range');
            const amountInput = document.getElementById('order-amount');
            
            amountRange.addEventListener('input', function() {
                amountInput.value = this.value;
                updateOrderDetails();
            });
            
            amountInput.addEventListener('input', function() {
                let value = parseInt(this.value);
                if (value < 1) value = 1;
                if (value > 1000) value = 1000;
                this.value = value;
                amountRange.value = value;
                updateOrderDetails();
            });
            
            // 交易类型切换
            document.querySelectorAll('[data-type]').forEach(btn => {
                btn.addEventListener('click', function() {
                    document.querySelectorAll('[data-type]').forEach(b => {
                        b.classList.remove('active');
                    });
                    this.classList.add('active');
                });
            });
            
            // 创建订单按钮
            document.getElementById('create-order-btn').addEventListener('click', createOrder);
            
            // 更新账户信息按钮
            document.getElementById('update-balance-btn').addEventListener('click', updateUserInfo);
            
            // 刷新订单按钮
            document.getElementById('refresh-orders-btn').addEventListener('click', loadAllOrders);
            
            // 应用筛选按钮
            document.getElementById('apply-filter-btn').addEventListener('click', loadAllOrders);
            
            // 注销按钮
            document.getElementById('logoutBtn').addEventListener('click', logoutUser);
        }
        
        // 显示指定页面
        function showPage(pageId) {
            // 隐藏所有页面
            document.querySelectorAll('.page-section').forEach(section => {
                section.classList.remove('active');
            });
            
            // 显示目标页面
            document.getElementById(`${pageId}-section`).classList.add('active');
            
            // 如果是订单页面，加载订单
            if (pageId === 'orders') {
                loadAllOrders();
            }
            // 如果是用户信息页面，加载用户信息
            else if (pageId === 'user-info') {
                loadUserInfo();
                loadUserOrders();
            }
            // 如果是仪表盘，加载数据
            else if (pageId === 'dashboard') {
                loadDashboardData();
            }
        }
        
        // 用户登录
        function loginUser() {
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;
            
            if (!username || !password) {
                showError('请输入用户名和密码');
                return;
            }
            
            // 显示加载效果
            const loginBtn = document.getElementById('loginBtn');
            const originalHtml = loginBtn.innerHTML;
            loginBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> 登录中...';
            loginBtn.disabled = true;
            
            // 调用登录API
            fetch(`${API_BASE_URL}/login/${encodeURIComponent(username)}/${encodeURIComponent(password)}`)
                .then(response => {
                    if (!response.ok) {
                        throw new Error('登录失败');
                    }
                    return response.text();
                })
                .then(data => {
                    // 登录成功
                    showSuccess('登录成功！');
                    
                    // 保存用户信息
                    const userId = data.match(/Hi, (.*)/)?.[1] || username;
                    currentUser = {
                        username: username,
                        userId: userId
                    };
                    
                    // 更新导航栏显示
                    document.getElementById('current-user').textContent = username;
                    
                    // 切换到仪表盘
                    setTimeout(() => {
                        document.getElementById('login-section').classList.remove('active');
                        document.getElementById('dashboard-section').classList.add('active');
                        
                        // 加载仪表盘数据
                        loadDashboardData();
                    }, 1000);
                })
                .catch(error => {
                    // 登录失败
                    showError(error.message || '登录失败，请检查用户名和密码');
                })
                .finally(() => {
                    // 重置按钮
                    loginBtn.innerHTML = originalHtml;
                    loginBtn.disabled = false;
                });
        }
        
        // 用户注册
        function registerUser(username, password) {
            // 显示加载效果
            const registerBtn = document.getElementById('showRegisterBtn');
            const originalHtml = registerBtn.innerHTML;
            registerBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> 注册中...';
            registerBtn.disabled = true;
            
            // 调用注册API
            fetch(`${API_BASE_URL}/register/${encodeURIComponent(username)}/${encodeURIComponent(password)}`)
                .then(response => {
                    if (!response.ok) {
                        throw new Error('注册失败');
                    }
                    return response.text();
                })
                .then(data => {
                    // 注册成功
                    showSuccess('注册成功！请登录');
                    
                    // 自动填充登录表单
                    document.getElementById('username').value = username;
                    document.getElementById('password').value = '';
                })
                .catch(error => {
                    // 注册失败
                    showError(error.message || '注册失败，用户名可能已被使用');
                })
                .finally(() => {
                    // 重置按钮
                    registerBtn.innerHTML = originalHtml;
                    registerBtn.disabled = false;
                });
        }
        
        // 加载仪表盘数据
        function loadDashboardData() {
            // 模拟加载用户信息
            document.getElementById('user-name').textContent = currentUser.username;
            document.getElementById('user-id').textContent = `ID: ${currentUser.userId}`;
            document.getElementById('available-energy').textContent = '750.5 kWh';
            document.getElementById('user-balance').textContent = '¥12,500.75';
            document.getElementById('active-orders').textContent = '3';
            document.getElementById('energy-price').textContent = '¥1.25';
            
            // 加载订单数据
            loadOrders();
        }
        
        // 加载订单数据
        function loadOrders() {
            const tableBody = document.getElementById('orders-table');
            tableBody.innerHTML = `
                <tr>
                    <td colspan="7" class="text-center py-5">
                        <i class="fas fa-spinner fa-spin me-2"></i>加载中...
                    </td>
                </tr>
            `;
            
            // 模拟API调用
            setTimeout(() => {
                tableBody.innerHTML = `
                    <tr>
                        <td>energy_order_100</td>
                        <td>出售</td>
                        <td>150</td>
                        <td>¥1.00</td>
                        <td><span class="status-badge status-created">已创建</span></td>
                        <td>2023-10-15 09:30</td>
                        <td>
                            <button class="btn btn-sm btn-outline-primary" data-order-id="energy_order_100">
                                查看
                            </button>
                        </td>
                    </tr>
                    <tr>
                        <td>energy_order_98</td>
                        <td>购买</td>
                        <td>200</td>
                        <td>¥0.98</td>
                        <td><span class="status-badge status-matched">已匹配</span></td>
                        <td>2023-10-14 14:20</td>
                        <td>
                            <button class="btn btn-sm btn-outline-primary" data-order-id="energy_order_98">
                                查看
                            </button>
                        </td>
                    </tr>
                    <tr>
                        <td>energy_order_95</td>
                        <td>出售</td>
                        <td>100</td>
                        <td>¥0.95</td>
                        <td><span class="status-badge status-finished">已完成</span></td>
                        <td>2023-10-12 10:15</td>
                        <td>
                            <button class="btn btn-sm btn-outline-primary" data-order-id="energy_order_95">
                                查看
                            </button>
                        </td>
                    </tr>
                `;
                
                // 绑定查看订单事件
                document.querySelectorAll('[data-order-id]').forEach(btn => {
                    btn.addEventListener('click', function() {
                        const orderId = this.getAttribute('data-order-id');
                        showOrderDetail(orderId);
                    });
                });
            }, 1000);
        }
        
        // 加载所有订单
        function loadAllOrders() {
            const tableBody = document.getElementById('all-orders-table');
            tableBody.innerHTML = `
                <tr>
                    <td colspan="8" class="text-center py-5">
                        <i class="fas fa-spinner fa-spin me-2"></i>加载中...
                    </td>
                </tr>
            `;
            
            // 模拟API调用
            setTimeout(() => {
                tableBody.innerHTML = `
                    <tr>
                        <td>energy_order_102</td>
                        <td>energy_user_15</td>
                        <td>-</td>
                        <td>150</td>
                        <td>¥1.00</td>
                        <td><span class="status-badge status-created">已创建</span></td>
                        <td>2023-10-16 10:30</td>
                        <td>
                            <button class="btn btn-sm btn-outline-primary" data-order-id="energy_order_102">
                                查看
                            </button>
                        </td>
                    </tr>
                    <tr>
                        <td>energy_order_101</td>
                        <td>energy_user_12</td>
                        <td>energy_user_08</td>
                        <td>80</td>
                        <td>¥1.05</td>
                        <td><span class="status-badge status-matched">已匹配</span></td>
                        <td>2023-10-15 15:45</td>
                        <td>
                            <button class="btn btn-sm btn-outline-primary" data-order-id="energy_order_101">
                                查看
                            </button>
                        </td>
                    </tr>
                    <tr>
                        <td>energy_order_100</td>
                        <td>energy_user_15</td>
                        <td>-</td>
                        <td>150</td>
                        <td>¥1.00</td>
                        <td><span class="status-badge status-created">已创建</span></td>
                        <td>2023-10-15 09:30</td>
                        <td>
                            <button class="btn btn-sm btn-outline-primary" data-order-id="energy_order_100">
                                查看
                            </button>
                        </td>
                    </tr>
                    <tr>
                        <td>energy_order_99</td>
                        <td>energy_user_10</td>
                        <td>energy_user_05</td>
                        <td>120</td>
                        <td>¥1.10</td>
                        <td><span class="status-badge status-finished">已完成</span></td>
                        <td>2023-10-14 16:20</td>
                        <td>
                            <button class="btn btn-sm btn-outline-primary" data-order-id="energy_order_99">
                                查看
                            </button>
                        </td>
                    </tr>
                `;
                
                // 绑定查看订单事件
                document.querySelectorAll('[data-order-id]').forEach(btn => {
                    btn.addEventListener('click', function() {
                        const orderId = this.getAttribute('data-order-id');
                        showOrderDetail(orderId);
                    });
                });
            }, 1000);
        }
        
        // 加载用户订单
        function loadUserOrders() {
            const tableBody = document.getElementById('user-orders-table');
            tableBody.innerHTML = `
                <tr>
                    <td colspan="6" class="text-center py-5">
                        <i class="fas fa-spinner fa-spin me-2"></i>加载中...
                    </td>
                </tr>
            `;
            
            // 模拟API调用
            setTimeout(() => {
                tableBody.innerHTML = `
                    <tr>
                        <td>energy_order_100</td>
                        <td>150</td>
                        <td>¥1.00</td>
                        <td><span class="status-badge status-created">已创建</span></td>
                        <td>2023-10-15 09:30</td>
                        <td>
                            <button class="btn btn-sm btn-outline-primary" data-order-id="energy_order_100">
                                查看
                            </button>
                        </td>
                    </tr>
                    <tr>
                        <td>energy_order_98</td>
                        <td>200</td>
                        <td>¥0.98</td>
                        <td><span class="status-badge status-matched">已匹配</span></td>
                        <td>2023-10-14 14:20</td>
                        <td>
                            <button class="btn btn-sm btn-outline-primary" data-order-id="energy_order_98">
                                查看
                            </button>
                        </td>
                    </tr>
                    <tr>
                        <td>energy_order_95</td>
                        <td>100</td>
                        <td>¥0.95</td>
                        <td><span class="status-badge status-finished">已完成</span></td>
                        <td>2023-10-12 10:15</td>
                        <td>
                            <button class="btn btn-sm btn-outline-primary" data-order-id="energy_order_95">
                                查看
                            </button>
                        </td>
                    </tr>
                `;
                
                // 绑定查看订单事件
                document.querySelectorAll('[data-order-id]').forEach(btn => {
                    btn.addEventListener('click', function() {
                        const orderId = this.getAttribute('data-order-id');
                        showOrderDetail(orderId);
                    });
                });
            }, 1000);
        }
        
        // 加载用户信息
        function loadUserInfo() {
            document.getElementById('user-name').textContent = currentUser.username;
            document.getElementById('user-id').textContent = `ID: ${currentUser.userId}`;
            document.getElementById('user-balance-info').textContent = '¥12,500.75';
            document.getElementById('user-available').textContent = '750.5 kWh';
            document.getElementById('user-role').textContent = '能源生产者';
            document.getElementById('total-orders').textContent = '12';
            document.getElementById('completed-orders').textContent = '8';
            document.getElementById('active-orders-count').textContent = '3';
            document.getElementById('total-energy').textContent = '1,250.5 kWh';
        }
        
        // 更新订单详情计算
        function updateOrderDetails() {
            const amount = parseFloat(document.getElementById('order-amount').value) || 0;
            const price = 1.25; // 从后端获取实际价格
            const fee = 0.01 * amount;
            const total = (price * amount) + fee;
            
            document.getElementById('order-price').textContent = `¥${price.toFixed(2)}`;
            document.getElementById('order-fee').textContent = `¥${fee.toFixed(2)}`;
            document.getElementById('order-total').textContent = `¥${total.toFixed(2)}`;
        }
        
        // 创建订单
        function createOrder() {
            const amount = document.getElementById('order-amount').value;
            
            if (!amount || amount < 1) {
                showError('请输入有效的能源数量');
                return;
            }
            
            // 显示加载效果
            const createBtn = document.getElementById('create-order-btn');
            const originalHtml = createBtn.innerHTML;
            createBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> 创建中...';
            createBtn.disabled = true;
            
            // 调用创建订单API
            fetch(`${API_BASE_URL}/createorder/${amount}`)
                .then(response => {
                    if (!response.ok) {
                        throw new Error('创建订单失败');
                    }
                    return response.text();
                })
                .then(data => {
                    // 创建成功
                    showSuccess('订单创建成功！');
                    
                    // 重新加载订单数据
                    loadOrders();
                    loadAllOrders();
                    loadUserOrders();
                })
                .catch(error => {
                    // 创建失败
                    showError(error.message || '创建订单失败，请重试');
                })
                .finally(() => {
                    // 重置按钮
                    createBtn.innerHTML = originalHtml;
                    createBtn.disabled = false;
                });
        }
        
        // 更新用户信息
        function updateUserInfo() {
            const updateBtn = document.getElementById('update-balance-btn');
            const originalHtml = updateBtn.innerHTML;
            updateBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> 更新中...';
            updateBtn.disabled = true;
            
            // 模拟API调用
            setTimeout(() => {
                // 更新成功
                showSuccess('账户信息已更新');
                
                // 重置按钮
                updateBtn.innerHTML = originalHtml;
                updateBtn.disabled = false;
            }, 1000);
        }
        
        // 显示订单详情
        function showOrderDetail(orderId) {
            // 设置订单详情
            document.getElementById('modal-order-id').textContent = orderId;
            document.getElementById('modal-party-a').textContent = 'energy_user_15';
            document.getElementById('modal-party-b').textContent = orderId.includes('98') ? 'energy_user_08' : '-';
            document.getElementById('modal-created-at').textContent = '2023-10-15 09:30:00';
            document.getElementById('modal-energy-amount').textContent = '150 kWh';
            document.getElementById('modal-energy-price').textContent = '¥1.00';
            document.getElementById('modal-transaction-fee').textContent = '¥1.50';
            document.getElementById('modal-total-amount').textContent = '¥150.00';
            
            // 设置订单状态
            let statusElement = document.getElementById('modal-order-status');
            if (orderId.includes('100')) {
                statusElement.textContent = '已创建';
                statusElement.className = 'status-badge status-created';
            } else if (orderId.includes('98')) {
                statusElement.textContent = '已匹配';
                statusElement.className = 'status-badge status-matched';
            } else {
                statusElement.textContent = '已完成';
                statusElement.className = 'status-badge status-finished';
            }
            
            // 显示模态框
            const modal = new bootstrap.Modal(document.getElementById('order-detail-modal'));
            modal.show();
        }
        
        // 用户注销
        function logoutUser() {
            // 调用注销API
            fetch(`${API_BASE_URL}/logout`)
                .then(response => {
                    if (!response.ok) {
                        throw new Error('注销失败');
                    }
                    return response.text();
                })
                .then(data => {
                    // 注销成功
                    showSuccess('您已成功注销');
                    
                    // 清除用户信息
                    currentUser = null;
                    
                    // 返回登录界面
                    setTimeout(() => {
                        document.querySelectorAll('.page-section').forEach(section => {
                            section.classList.remove('active');
                        });
                        document.getElementById('login-section').classList.add('active');
                        
                        // 清空表单
                        document.getElementById('username').value = '';
                        document.getElementById('password').value = '';
                        
                        // 更新导航栏显示
                        document.getElementById('current-user').textContent = '未登录';
                    }, 1000);
                })
                .catch(error => {
                    // 注销失败
                    showError(error.message || '注销失败，请重试');
                });
        }
        
        // 显示错误消息
        function showError(message) {
            const element = document.getElementById('error-message');
            element.textContent = message;
            element.style.display = 'block';
            
            // 3秒后自动隐藏
            setTimeout(() => {
                element.style.display = 'none';
            }, 3000);
        }
        
        // 显示成功消息
        function showSuccess(message) {
            const element = document.getElementById('success-message');
            element.textContent = message;
            element.style.display = 'block';
            
            // 3秒后自动隐藏
            setTimeout(() => {
                element.style.display = 'none';
            }, 3000);
        }
    </script>
</body>
</html>