from pyspark import SparkContext, SparkConf

if __name__ == "__main__":
    conf = SparkConf().setAppName("word count").setMaster(
        "spark://ip-172-31-95-231.ec2.internal:7077")
    sc = SparkContext(conf=conf)

    lines = sc.textFile("data/book.txt")

    words = lines.flatMap(lambda line: line.split())

    wordCounts = words.countByValue()

    for word, count in wordCounts.items():
        print("{}: {}, ".format(word, count), end="")
