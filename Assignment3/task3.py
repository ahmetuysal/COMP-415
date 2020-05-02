from pyspark import SparkContext, SparkConf

if __name__ == "__main__":
    conf = SparkConf().setAppName("words lower").setMaster(
        "spark://ip-172-31-95-231.ec2.internal:7077")
    sc = SparkContext(conf=conf)

    lines = sc.textFile("data/book.txt")

    # Using this is simple and safer (due to memory requirements) but this does not allow
    # naming of the output files
    # lines.map(str.lower).saveAsTextFile("words_lower.txt")

    lines_to_lower = lines.map(str.lower).collect()

    with open("words_lower.txt", "w") as output_file:
        for line in lines_to_lower:
            output_file.write(line + "\n")
