#import <Foundation/Foundation.h>

int skillview_archive_move(const char *source, const char *destination) {
 @autoreleasepool {
  NSURL *from = [NSURL fileURLWithPath:[NSString stringWithUTF8String:source]];
  NSURL *to = [NSURL fileURLWithPath:[NSString stringWithUTF8String:destination]];
  return [[NSFileManager defaultManager] moveItemAtURL:from toURL:to error:nil] ? 1 : 0;
 }
}
