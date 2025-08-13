<?php
 if ($_SERVER["REQUEST_METHOD"] == "GET" && !empty($_GET["id"])) {
    $data = array(
        "ID" => $_GETT["id"],
    );    
    $url = "http://localhost:8080/delete/" . $_GET["id"];
    $ch = curl_init($url);
    curl_setopt($ch, CURLOPT_CUSTOMREQUEST, "GET");
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    $response = curl_exec($ch);
    curl_close($ch);
    echo "<h2>链码返回结果：</h2>";
    echo "<pre>" . htmlspecialchars($response) . "</pre>";
 }
 ?>
<!-- HTML 表单 -->
<!DOCTYPE html>
 <html>
 <head>
    <meta charset="UTF-8">
    <title>新增资产</title>
    <style>
        body {
            font-family: Arial;
            margin: 40px;
        }
        form {
            margin-bottom: 30px;
        }
        label {
            display: inline-block;
            width: 150px;
        }
        input[type="text"], input[type="number"] {
            width: 200px;
            padding: 5px;
        }
        input[type="submit"], .button-link {
            padding: 10px 20px;
            font-size: 16px;
            margin-top: 10px;
            margin-right: 10px;
            cursor: pointer;
        }
        .button-link {
            background-color: #007bff;
            color: white;
            text-decoration: none;
            border-radius: 5px;
            display: inline-block;
        }
        .button-link:hover {
            background-color: #0056b3;
        }
    </style>
 </head>
 <body>
 <h1>添加 Asset</h1>
 <form method="GET">
    <label>ID:</label>
    <input type="text" name="id" required><br><br>
    <input type="submit" value="删除">
 </form>
 <!-- 添加按钮跳转到 all_assets.php -->
 <a href="index.php" class="button-link">回到主页</a>
 </body>
 </html>
