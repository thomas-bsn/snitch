from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.benchmark_dataset_created import BenchmarkDatasetCreated
from ...models.benchmark_dataset_input import BenchmarkDatasetInput
from ...models.error import Error
from typing import cast



def _get_kwargs(
    *,
    body: BenchmarkDatasetInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/benchmarks/datasets",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> BenchmarkDatasetCreated | Error:
    if response.status_code == 201:
        response_201 = BenchmarkDatasetCreated.from_dict(response.json())



        return response_201

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[BenchmarkDatasetCreated | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    body: BenchmarkDatasetInput,
    origin: str,
    x_csrf_token: str,

) -> Response[BenchmarkDatasetCreated | Error]:
    """  Administrateurs uniquement. Crée un corpus de clips partagés : clips de test (éventuellement
    certifiés relus en entier) et clips de référence (enrôlement des Personnes pour l'empreinte). La
    vérité terrain des voix est la validation humaine, figée à la création. Les étapes mesurables en
    sont déduites (stages). Refus : benchmark_clip_duplicate, benchmark_clip_invalid (inconnu ou privé),
    benchmark_verified_not_test, benchmark_dataset_unusable (aucune étape mesurable),
    benchmark_dataset_name_taken.

    Args:
        origin (str):
        x_csrf_token (str):
        body (BenchmarkDatasetInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BenchmarkDatasetCreated | Error]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    body: BenchmarkDatasetInput,
    origin: str,
    x_csrf_token: str,

) -> BenchmarkDatasetCreated | Error | None:
    """  Administrateurs uniquement. Crée un corpus de clips partagés : clips de test (éventuellement
    certifiés relus en entier) et clips de référence (enrôlement des Personnes pour l'empreinte). La
    vérité terrain des voix est la validation humaine, figée à la création. Les étapes mesurables en
    sont déduites (stages). Refus : benchmark_clip_duplicate, benchmark_clip_invalid (inconnu ou privé),
    benchmark_verified_not_test, benchmark_dataset_unusable (aucune étape mesurable),
    benchmark_dataset_name_taken.

    Args:
        origin (str):
        x_csrf_token (str):
        body (BenchmarkDatasetInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BenchmarkDatasetCreated | Error
     """


    return sync_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    body: BenchmarkDatasetInput,
    origin: str,
    x_csrf_token: str,

) -> Response[BenchmarkDatasetCreated | Error]:
    """  Administrateurs uniquement. Crée un corpus de clips partagés : clips de test (éventuellement
    certifiés relus en entier) et clips de référence (enrôlement des Personnes pour l'empreinte). La
    vérité terrain des voix est la validation humaine, figée à la création. Les étapes mesurables en
    sont déduites (stages). Refus : benchmark_clip_duplicate, benchmark_clip_invalid (inconnu ou privé),
    benchmark_verified_not_test, benchmark_dataset_unusable (aucune étape mesurable),
    benchmark_dataset_name_taken.

    Args:
        origin (str):
        x_csrf_token (str):
        body (BenchmarkDatasetInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BenchmarkDatasetCreated | Error]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    body: BenchmarkDatasetInput,
    origin: str,
    x_csrf_token: str,

) -> BenchmarkDatasetCreated | Error | None:
    """  Administrateurs uniquement. Crée un corpus de clips partagés : clips de test (éventuellement
    certifiés relus en entier) et clips de référence (enrôlement des Personnes pour l'empreinte). La
    vérité terrain des voix est la validation humaine, figée à la création. Les étapes mesurables en
    sont déduites (stages). Refus : benchmark_clip_duplicate, benchmark_clip_invalid (inconnu ou privé),
    benchmark_verified_not_test, benchmark_dataset_unusable (aucune étape mesurable),
    benchmark_dataset_name_taken.

    Args:
        origin (str):
        x_csrf_token (str):
        body (BenchmarkDatasetInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BenchmarkDatasetCreated | Error
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
