#!/usr/bin/env python3

import os, re

res_folder = 'web'
res_file = 'webresources.go'
re_resources = re.compile('^.+\.(?P<ext>html|css|js)$', re.IGNORECASE)

res_file_header = """package main

// GENERATED FILE: Append here all the resources to be exposed as variables
// webresources["index.html"] = etc...
// webresources["css/style.css"] = etc...

func setVariables() {"""

res_file_footer = """
return }
"""
if __name__ == '__main__':
    print('Extracting web resources and storing them in a go map\n')
    
    # use text file, utf-8, line buffering
    with open(res_file, 'tw', encoding='utf-8', buffering=1) as res_f:
        
        res_f.write(res_file_header)
        
        for root, dirs, files in  os.walk(res_folder, topdown=True):
            rel_root = os.path.join(root[len(res_folder) + 1:]).lower()
            for file_name in files:
                if(not re_resources.search(file_name)):
                     continue

                # webresources["css/reset.css"] = ` Ensure that \u0060 character replaces the back quote `
                f_var = os.path.normpath(os.path.join(rel_root,file_name)).replace('\\', '/')
                res_f.write('webresources["%s"] = `' % f_var)
                with open(os.path.join(root, file_name), 'r', encoding='utf-8') as f:
                    for l in f:
                        # parse and add file line
                        res_f.write(l.replace('`', '\\u0060'))
                res_f.write('`\n') # closing quote
        
        res_f.write(res_file_footer)

