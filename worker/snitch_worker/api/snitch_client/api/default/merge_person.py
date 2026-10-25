from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.person_correction_result import PersonCorrectionResult
from ...models.person_merge_input import PersonMergeInput
from typing import cast



def _get_kwargs(
    id: int,
    *,
    body: PersonMergeInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/persons/{id}/merge".format(id=quote(str(id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | PersonCorrectionResult:
    if response.status_code == 200:
        response_200 = PersonCorrectionResult.from_dict(response.json())



        return response_200

    if response.status_code == 409:
        response_409 = Error.from_dict(response.json())



        return response_409

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | PersonCorrectionResult]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    *,
    client: AuthenticatedClient,
    body: PersonMergeInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | PersonCorrectionResult]:
    """  Utilisateurs connectés. Rattache à la cible les attributions validées (la première validation fait
    foi, seule la Personne change), les Échantillons vocaux et les désignations de piste micro de la
    Personne absorbée, qui disparaît ; le rapprochement est recalculé sans worker. Une entrée
    d'historique (action merge) par voix.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (PersonMergeInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PersonCorrectionResult]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    *,
    client: AuthenticatedClient,
    body: PersonMergeInput,
    origin: str,
    x_csrf_token: str,

) -> Error | PersonCorrectionResult | None:
    """  Utilisateurs connectés. Rattache à la cible les attributions validées (la première validation fait
    foi, seule la Personne change), les Échantillons vocaux et les désignations de piste micro de la
    Personne absorbée, qui disparaît ; le rapprochement est recalculé sans worker. Une entrée
    d'historique (action merge) par voix.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (PersonMergeInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PersonCorrectionResult
     """


    return sync_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: int,
    *,
    client: AuthenticatedClient,
    body: PersonMergeInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | PersonCorrectionResult]:
    """  Utilisateurs connectés. Rattache à la cible les attributions validées (la première validation fait
    foi, seule la Personne change), les Échantillons vocaux et les désignations de piste micro de la
    Personne absorbée, qui disparaît ; le rapprochement est recalculé sans worker. Une entrée
    d'historique (action merge) par voix.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (PersonMergeInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PersonCorrectionResult]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    *,
    client: AuthenticatedClient,
    body: PersonMergeInput,
    origin: str,
    x_csrf_token: str,

) -> Error | PersonCorrectionResult | None:
    """  Utilisateurs connectés. Rattache à la cible les attributions validées (la première validation fait
    foi, seule la Personne change), les Échantillons vocaux et les désignations de piste micro de la
    Personne absorbée, qui disparaît ; le rapprochement est recalculé sans worker. Une entrée
    d'historique (action merge) par voix.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (PersonMergeInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PersonCorrectionResult
     """


    return (await asyncio_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
