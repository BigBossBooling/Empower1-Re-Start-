# This file makes Python treat the `ire` directory as a sub-package of `empower1`.

# You can also import key components from the IRE module here for easier access, e.g.:
# from .redistribution import RedistributionEngine
# from .ai_model import IREDecisionModel

DEFAULT_TAX_RATE = 0.09 # 9% tax rate as mentioned in README
DEFAULT_STIMULUS_THRESHOLD = 1000 # Example: users with wealth below this might receive stimulus
MIN_TRANSACTION_FOR_TAX = 10 # Example: tax only applies to transactions above this amount

print("Intelligent Redistribution Engine (IRE) module loaded.")
