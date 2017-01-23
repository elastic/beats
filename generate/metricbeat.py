import helper

if __name__ == "__main__":

    parser = helper.get_parser()
    args = parser.parse_args()
    helper.generate_beat("metricbeat", args)

