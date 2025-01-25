set seed 9999 
local n = 10000 

local annual_vacc_prob = 0.6
local vacc_peroid      = 90
local ve               = 0.6
local flu_peroid       = 30*4
local flu_prob         = 0.2
local nonflu_prob      = 0.05
local test_prob        = 0.1

clear 
set obs `n' 

//Weibull (proportional hazards) variates with shape a,
//                     scale b, and location g

gen vacct=int(rweibull(1,1/(`annual_vacc_prob'/`vacc_peroid')))
gen vacc=  vacct > `vacc_peroid' | vacct ==0
local flu_rate =  1/((`flu_prob'/`flu_peroid')*2.0)
gen flut=int(rweibull(1.7,`flu_rate'))
replace flut =int(rweibull(1.7,`flu_rate'*`ve'))

gen flu= flut<`flu_peroid'
gen nonflut=int(rweibull(1.7,1/((`nonflu_prob'/`flu_peroid')*2.0)))
gen nonflu=nonflut < `flu_peroid'
gen tested=runiform() < `test_prob' if flu+nonflu >0
gen invalid= flu>0 & flut < vacct
