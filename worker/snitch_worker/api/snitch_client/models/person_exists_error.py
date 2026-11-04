from enum import StrEnum

class PersonExistsError(StrEnum):
    PERSON_EXISTS = "person_exists"

    def __str__(self) -> str:
        return str(self.value)
