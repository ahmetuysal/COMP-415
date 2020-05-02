from pyspark import SparkContext, SparkConf

if __name__ == "__main__":
    conf = SparkConf().setAppName("lines dash").setMaster(
        "spark://ip-172-31-95-231.ec2.internal:7077")
    sc = SparkContext(conf=conf)

    lines = sc.textFile("data/book.txt")

    lines_dashed = lines.map(lambda line: line.replace(" ", "-")).collect()

    with open("words-.txt", "w") as output_file:
        for line in lines_dashed:
            output_file.write(line + "\n")
