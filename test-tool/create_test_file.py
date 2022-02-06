import argparse
import os

def create_file(count, size, name, dst_dir):
    dst = os.path.join(dst_dir, name)
    print('writing {} lines each is {} bytes dst {}'.format(count, size, dst))
    large_str = "a" * size
    with open(dst, 'w') as f:
        for i in range(count):
            f.write('{"a":"' + large_str + '"}\n')

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Create test file for lumigo-io/awsbeats')
    parser.add_argument('--type', nargs='+',
                    choices=['1k800kb', '1k500kb', '10k100kb', '1m1kb'],
                    help='Test file type')
    parser.add_argument('--dir', nargs='+',
                    help='File destination directory')
    
    config_dct = vars(parser.parse_args())
    test_type = config_dct['type'][0]
    test_dir = config_dct['dir'][0]
    print(test_dir)
    if test_type == '1k800kb':
        create_file(1000, 800000, test_type, test_dir)
    elif test_type == '1k500kb':
        create_file(1000, 500000, test_type, test_dir)
    elif test_type == '10k10kb':
        create_file(10000, 10000, test_type, test_dir)
    elif test_type == '1m1kb':
        create_file(1000000, 1000, test_type, test_dir)
    else:
        print('doing nothing got {}'.format(test_type))