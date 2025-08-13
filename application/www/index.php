 <!DOCTYPE html>
 <html>

 <head>
     <meta charset="UTF-8">
     <title>资产管理系统</title>
     <style>
         body {
             font-family: Arial, sans-serif;
             background-color: #f2f2f2;
             text-align: center;
             padding-top: 80px;
         }

         h1 {
             color: #333;
         }

         .menu {
             margin-top: 40px;
         }

         .menu a {
             display: inline-block;
             margin: 20px;
             padding: 20px 40px;
             text-decoration: none;
             font-size: 18px;
             color: white;
             background-color: #007bff;
             border-radius: 10px;
             transition: background-color 0.3s ease;
         }

         .menu a:hover {
             background-color: #0056b3;
         }
     </style>
 </head>

 <body>
     <h1>欢迎使用资产管理系统</h1>
     <div class="menu">
         <a href="all_assets.php">查看所有资产</a>
         <a href="post_asset.php">新增资产</a>
         <a href="update_asset.php">更新资产</a>
         <a href="delete_asset.php">删除资产</a>
     </div>
 </body>

 </html>