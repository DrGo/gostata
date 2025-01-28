# Stata dta writer and script runner

Package stata writes data into a Stata 113 format (readable by any Stata version higher than 7)
Source for format info https://www.stata.com/help.cgi?dta_113

## Limitations
only supports Little Endian encoding, but all Stata flavours are capable of reading it.
The package does not do much validation. It is up to the user to ensure that the supplied data
## Reference
https://www.stata.com/help.cgi?dta_113

//TODO 
- clean up interface
- expose all relevant constants eg missing and max size 
- add documentation. 


