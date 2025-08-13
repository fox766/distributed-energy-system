 <?php
  $response = file_get_contents("http://localhost:8080/assets");
  $data = json_decode($response, true);
  ?>
 <!DOCTYPE html>
 <html>

 <head>
   <meta charset="UTF-8">
   <title>资产列表</title>
   <style>
     body {
       font-family: Arial, sans-serif;
       margin: 40px;
       background-color: #f4f4f4;
     }

     h1 {
       color: #333;
     }

     table {
       border-collapse: collapse;
       width: 100%;
       background-color: #fff;
       box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
     }

     th,
     td {
       padding: 12px 15px;
       border: 1px solid #ddd;
       text-align: center;
     }

     th {
       background-color: #007bff;
       color: white;
     }

     tr:nth-child(even) {
       background-color: #f9f9f9;
     }

     tr:hover {
       background-color: #e0f0ff;
     }
   </style>
 </head>

 <body>
   <h1>资产列表</h1>
   <table>
     <tr>
       <th>ID</th>
       <th>颜色</th>
       <th>尺寸</th>
       <th>拥有者</th>
       <th>估价</th>
     </tr>
     <?php if (is_array($data)) : ?>
       <?php foreach ($data as $asset): ?>
         <tr>
           <td><?= htmlspecialchars($asset["ID"]) ?></td>
           <td><?= htmlspecialchars($asset["Color"]) ?></td>
           <td><?= htmlspecialchars($asset["Size"]) ?></td>
           <td><?= htmlspecialchars($asset["Owner"]) ?></td>
           <td><?= htmlspecialchars($asset["AppraisedValue"]) ?></td>
         </tr>
       <?php endforeach; ?>
     <?php else: ?>
       <tr>
         <td colspan="5">未获取到有效数据</td>
       </tr>
     <?php endif; ?>
   </table>
 </body>

 </html>