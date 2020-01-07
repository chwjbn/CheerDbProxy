# CheerDbProxy
基于kingshard改造的数据库运维权限控制中间件，管理端不开源，做了如下改造：
- 配置存在redis，管理端将权限配置写入redis，dbproxy定期（20秒）读取redis配置并刷新
- 优化后端节点故障错误机制，保持永活
- 增加show databases/use 命令权限控制

管理端效果图

![image](https://raw.githubusercontent.com/chwjbn/CheerDbProxy/master/Doc/1.jpg)
![image](https://raw.githubusercontent.com/chwjbn/CheerDbProxy/master/Doc/2.jpg)
![image](https://raw.githubusercontent.com/chwjbn/CheerDbProxy/master/Doc/3.jpg)
![image](https://raw.githubusercontent.com/chwjbn/CheerDbProxy/master/Doc/4.jpg)
