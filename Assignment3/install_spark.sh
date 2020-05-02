#!/usr/bin/env bash

# install java 8
sudo yum install java-1.8.0-openjdk

# install spark
curl -O https://downloads.apache.org/spark/spark-2.4.5/spark-2.4.5-bin-hadoop2.7.tgz
tar xvf spark-2.4.5-bin-hadoop2.7.tgz
sudo mv spark-2.4.5-bin-hadoop2.7/ /opt/spark 
rm spark-2.4.5-bin-hadoop2.7.tgz

# add spark to path variables
echo 'export SPARK_HOME=/opt/spark' >> ~/.bashrc 
echo 'export PATH=$PATH:$SPARK_HOME/bin:$SPARK_HOME/sbin' >> ~/.bashrc

# install miniconda
wget https://repo.anaconda.com/miniconda/Miniconda3-latest-Linux-x86_64.sh -O ~/miniconda.sh
bash ~/miniconda.sh -b -p $HOME/miniconda

eval "$(~/miniconda/bin/conda shell.bash hook)"

# install pyspark and matplotlib
conda init
conda install pyspark
conda install matplotlib

source ~/.bashrc

# set default python version to use with pyspark
echo 'export PYSPARK_PYTHON=~/miniconda/bin/python' >> $SPARK_HOME/conf/spark-env.sh

# download data files
fileid="13jvI3iEZZhjLk3AS9JY3rP4Dsqc4RhbN"
filename="numbers.zip"
curl -c ./cookie -s -L "https://drive.google.com/uc?export=download&id=${fileid}" > /dev/null
curl -Lb ./cookie "https://drive.google.com/uc?export=download&confirm=`awk '/download/ {print $NF}' ./cookie`&id=${fileid}" -o ${filename}

bookid="14syoBeUAb-e8tjSbCbCTLQvBiAZOmKY8"
bookname="book.txt"
curl -c ./cookie -s -L "https://drive.google.com/uc?export=download&id=${bookid}" > /dev/null
curl -Lb ./cookie "https://drive.google.com/uc?export=download&confirm=`awk '/download/ {print $NF}' ./cookie`&id=${bookid}" -o ${bookname}

mkdir data
mv book.txt data
sudo unzip numbers.zip -d data

source ~/.bashrc
