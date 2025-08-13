<?php
 if ($_SERVER["REQUEST_METHOD"] == "POST") {
    $data = array(
        "ID" => $_POST["id"],
        "Color" => $_POST["color"],
        "Size" => (int)$_POST["size"],
        "Owner" => $_POST["owner"],
        "AppraisedValue" => (int)$_POST["value"]
    );
    $jsonData = json_encode($data);
    $url = "http://localhost:8080/assetUpdate"; 
    $ch = curl_init($url);
    curl_setopt($ch, CURLOPT_CUSTOMREQUEST, "POST");
    curl_setopt($ch, CURLOPT_POSTFIELDS, $jsonData);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_HTTPHEADER, [
        'Content-Type: application/json',
        'Content-Length: ' . strlen($jsonData)
    ]);
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
    <title>更新资产</title>
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
 <h1>更新 Asset</h1>
 <form method="POST">
    <label>ID:</label>
    <input type="text" name="id" required><br><br>
    <label>Color:</label>
    <input type="text" name="color" required><br><br>
    <label>Size:</label>
    <input type="number" name="size" required><br><br>
    <label>Owner:</label>
    <input type="text" name="owner" required><br><br>
    <label>Appraised Value:</label>
    <input type="number" name="value" required><br><br>
    <input type="submit" value="提交">
 </form>
 <!-- 添加按钮跳转到 all_assets.php -->
 <a href="index.php" class="button-link">回到主页</a>
 </body>
 </html>
